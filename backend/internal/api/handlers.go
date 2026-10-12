// Package api exposes the engine over HTTP.
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"authgate/internal/authz"
	"authgate/internal/domain"
	"authgate/internal/store"
)

// Config locates files the API serves. Paths are relative to the backend dir.
type Config struct {
	FeaturesDir  string // e.g. "features"
	ReportPath   string // e.g. "../bdd-report.json"
	EnableRunner bool   // POST /api/bdd/run
}

type server struct {
	st  *store.Store
	cfg Config
}

// NewRouter builds the HTTP handler.
func NewRouter(st *store.Store, cfg Config) http.Handler {
	s := &server{st: st, cfg: cfg}
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))
	r.Route("/api", func(r chi.Router) {
		r.Post("/authorize", s.authorize)
		r.Get("/seed", s.seed)
		r.Get("/accounts/{id}", s.account)
		r.Get("/cards/{id}/transactions", s.cardTransactions)
		r.Post("/reset", s.reset)
		r.Get("/features", s.features)
		r.Get("/bdd-report", s.report)
		if cfg.EnableRunner {
			r.Post("/bdd/run", s.runBDD)
		}
		r.Get("/config", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]bool{"bddRunner": cfg.EnableRunner})
		})
	})
	return r
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

type authorizeBody struct {
	CardID      string `json:"cardId"`
	MerchantID  string `json:"merchantId"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
}

func (s *server) authorize(w http.ResponseWriter, r *http.Request) {
	var b authorizeBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if b.AmountCents <= 0 {
		writeErr(w, http.StatusBadRequest, "amountCents must be positive")
		return
	}
	res, err := authz.Authorize(s.st, authz.Request{CardID: b.CardID, MerchantID: b.MerchantID, Amount: b.AmountCents, Currency: b.Currency})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *server) seed(w http.ResponseWriter, _ *http.Request) {
	var out struct {
		Accounts  []domain.Account  `json:"accounts"`
		Cards     []domain.Card     `json:"cards"`
		Merchants []domain.Merchant `json:"merchants"`
	}
	s.st.View(func(t *store.Tx) {
		out.Accounts, out.Cards, out.Merchants = t.Accounts(), t.Cards(), t.Merchants()
	})
	writeJSON(w, http.StatusOK, out)
}

func (s *server) account(w http.ResponseWriter, r *http.Request) {
	var out struct {
		Account      domain.Account       `json:"account"`
		Transactions []domain.Transaction `json:"transactions"`
	}
	found := false
	s.st.View(func(t *store.Tx) {
		out.Account, found = t.Account(chi.URLParam(r, "id"))
		if found {
			out.Transactions = limit(t.AccountTransactions(out.Account.ID), 20)
		}
	})
	if !found {
		writeErr(w, http.StatusNotFound, "account not found")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) cardTransactions(w http.ResponseWriter, r *http.Request) {
	var txns []domain.Transaction
	found := false
	s.st.View(func(t *store.Tx) {
		_, found = t.Card(chi.URLParam(r, "id"))
		txns = limit(t.CardTransactions(chi.URLParam(r, "id")), 20)
	})
	if !found {
		writeErr(w, http.StatusNotFound, "card not found")
		return
	}
	writeJSON(w, http.StatusOK, txns)
}

func limit(x []domain.Transaction, n int) []domain.Transaction {
	if x == nil {
		return []domain.Transaction{}
	}
	if len(x) > n {
		return x[:n]
	}
	return x
}

func (s *server) reset(w http.ResponseWriter, _ *http.Request) {
	s.st.Reset()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) features(w http.ResponseWriter, _ *http.Request) {
	files, _ := filepath.Glob(filepath.Join(s.cfg.FeaturesDir, "*.feature"))
	sort.Strings(files)
	out := []string{}
	for _, f := range files {
		if b, err := os.ReadFile(f); err == nil {
			out = append(out, string(b))
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) report(w http.ResponseWriter, _ *http.Request) {
	b, err := os.ReadFile(s.cfg.ReportPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("null"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

// runBDD shells out to the godog suite. The suite exits non-zero when scenarios
// fail (that is the demo), so the exit status is not treated as an error; the
// report file is the source of truth.
func (s *server) runBDD(w http.ResponseWriter, _ *http.Request) {
	cmd := exec.Command("go", "test", ".", "-run", "TestFeatures", "-count=1")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	b, rerr := os.ReadFile(s.cfg.ReportPath)
	if rerr != nil {
		writeErr(w, http.StatusInternalServerError, "suite did not produce a report: "+strings.TrimSpace(string(out))+" "+errString(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
