package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authgate/internal/api"
	"authgate/internal/authz"
	"authgate/internal/store"
)

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
	return rec
}

func TestAuthorizeAndReset(t *testing.T) {
	h := api.NewRouter(store.New(), api.Config{FeaturesDir: "../../features", ReportPath: "none.json"})

	rec := post(t, h, "/api/authorize", `{"cardId":"CARD-002","merchantId":"MERCH-COFFEE","amountCents":1250,"currency":"USD"}`)
	var res authz.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Approved || res.Reason != "CARD_FROZEN" {
		t.Fatalf("got %+v", res)
	}

	rec = post(t, h, "/api/authorize", `{"cardId":"CARD-001","merchantId":"MERCH-COFFEE","amountCents":1250,"currency":"USD"}`)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !res.Approved || res.NewBalance != 48750 {
		t.Fatalf("got %+v", res)
	}

	post(t, h, "/api/reset", "")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/accounts/ACC-001", nil))
	if !strings.Contains(rec.Body.String(), `"balance":50000`) {
		t.Fatalf("reset did not restore balance: %s", rec.Body.String())
	}
}

func TestBadRequest(t *testing.T) {
	h := api.NewRouter(store.New(), api.Config{})
	if rec := post(t, h, "/api/authorize", `{`); rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestRunnerDisabledByDefault(t *testing.T) {
	h := api.NewRouter(store.New(), api.Config{})
	if rec := post(t, h, "/api/bdd/run", ""); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code %d", rec.Code)
	}
}
