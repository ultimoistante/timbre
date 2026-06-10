package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ultimoistante/timbre/internal/config"
	"github.com/ultimoistante/timbre/internal/db"
)

// newTestServer builds a Server backed by a temp-dir sqlite DB.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	cfg, err := config.Load() // reads env; override DataDir below
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	cfg.DataDir = t.TempDir()
	cfg.DBDriver = "sqlite"
	cfg.DBDSN = cfg.DataDir + "/test.db"

	gdb, err := db.Open(cfg)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	return New(cfg, gdb)
}

func doJSON(t *testing.T, srv *Server, method, path, body, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestOnboardingLoginMeFlow(t *testing.T) {
	srv := newTestServer(t)

	// Status: needs onboarding.
	rec := doJSON(t, srv, http.MethodGet, "/api/onboarding", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status code %d", rec.Code)
	}
	var st struct {
		NeedsOnboarding bool `json:"needsOnboarding"`
	}
	json.Unmarshal(rec.Body.Bytes(), &st)
	if !st.NeedsOnboarding {
		t.Fatal("expected needsOnboarding=true")
	}

	// Onboard admin.
	rec = doJSON(t, srv, http.MethodPost, "/api/onboarding", `{"username":"admin","password":"pw12345"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("onboarding code %d: %s", rec.Code, rec.Body)
	}
	var onboard struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			ID   uint   `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	json.Unmarshal(rec.Body.Bytes(), &onboard)
	if onboard.User.Role != "admin" {
		t.Fatalf("first user must be admin, got %q", onboard.User.Role)
	}
	if onboard.AccessToken == "" {
		t.Fatal("expected access token")
	}

	// Second onboarding must fail.
	rec = doJSON(t, srv, http.MethodPost, "/api/onboarding", `{"username":"x","password":"y"}`, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("second onboarding should conflict, got %d", rec.Code)
	}

	// /me without token -> 401.
	rec = doJSON(t, srv, http.MethodGet, "/api/me", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("/me unauth should be 401, got %d", rec.Code)
	}

	// /me with token -> 200.
	rec = doJSON(t, srv, http.MethodGet, "/api/me", "", onboard.AccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("/me code %d: %s", rec.Code, rec.Body)
	}

	// Login with correct creds.
	rec = doJSON(t, srv, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"pw12345"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login code %d: %s", rec.Code, rec.Body)
	}

	// Login with wrong password.
	rec = doJSON(t, srv, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"nope"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login should be 401, got %d", rec.Code)
	}
}
