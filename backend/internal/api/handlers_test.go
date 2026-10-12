package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authgate/internal/api"
	"authgate/internal/auth"
	"authgate/internal/authz"
	"authgate/internal/store"
)

func newRouter(cfg api.Config) http.Handler {
	return api.NewRouter(store.New(), auth.New(), cfg)
}

func send(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func login(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := send(h, http.MethodPost, "/api/login", `{"username":"alice","password":"wonderland1"}`, "")
	var sess auth.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &sess); err != nil || sess.Token == "" {
		t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}
	return sess.Token
}

func TestAuthorizeAndReset(t *testing.T) {
	h := newRouter(api.Config{FeaturesDir: "../../features", ReportPath: "none.json"})
	tok := login(t, h)
	post := func(path, body string) *httptest.ResponseRecorder { return send(h, http.MethodPost, path, body, tok) }

	rec := post("/api/authorize", `{"cardId":"CARD-002","merchantId":"MERCH-COFFEE","amountCents":1250,"currency":"USD"}`)
	var res authz.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Approved || res.Reason != "CARD_FROZEN" {
		t.Fatalf("got %+v", res)
	}

	rec = post("/api/authorize", `{"cardId":"CARD-001","merchantId":"MERCH-COFFEE","amountCents":1250,"currency":"USD"}`)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !res.Approved || res.NewBalance != 48750 {
		t.Fatalf("got %+v", res)
	}

	post("/api/reset", "")
	rec = send(h, http.MethodGet, "/api/accounts/ACC-001", "", tok)
	if !strings.Contains(rec.Body.String(), `"balance":50000`) {
		t.Fatalf("reset did not restore balance: %s", rec.Body.String())
	}
}

func TestBadRequest(t *testing.T) {
	h := newRouter(api.Config{})
	tok := login(t, h)
	if rec := send(h, http.MethodPost, "/api/authorize", `{`, tok); rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestRunnerDisabledByDefault(t *testing.T) {
	h := newRouter(api.Config{})
	tok := login(t, h)
	if rec := send(h, http.MethodPost, "/api/bdd/run", "", tok); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestAuthRequired(t *testing.T) {
	h := newRouter(api.Config{})
	if rec := send(h, http.MethodGet, "/api/seed", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: code %d", rec.Code)
	}
	tok := login(t, h)
	if rec := send(h, http.MethodGet, "/api/seed", "", tok); rec.Code != http.StatusOK {
		t.Fatalf("with token: code %d", rec.Code)
	}
	send(h, http.MethodPost, "/api/logout", "", tok)
	if rec := send(h, http.MethodGet, "/api/seed", "", tok); rec.Code != http.StatusUnauthorized {
		t.Fatalf("after logout: code %d", rec.Code)
	}
}

func TestLoginFailures(t *testing.T) {
	h := newRouter(api.Config{})
	rec := send(h, http.MethodPost, "/api/login", `{"username":"alice","password":"nope"}`, "")
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "INVALID_CREDENTIALS") {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	rec = send(h, http.MethodPost, "/api/login", `{"username":"","password":""}`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rec.Code)
	}
}
