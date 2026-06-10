package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
)

// onboardAdmin creates the admin and returns its access token.
func onboardAdmin(t *testing.T, srv *Server) string {
	t.Helper()
	rec := doJSON(t, srv, http.MethodPost, "/api/onboarding", `{"username":"admin","password":"pw12345"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("onboarding failed: %d %s", rec.Code, rec.Body)
	}
	var out struct {
		AccessToken string `json:"accessToken"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	return out.AccessToken
}

// makeUser inserts a user directly and returns an access token for it.
func makeUser(t *testing.T, srv *Server, name string) (*models.User, string) {
	t.Helper()
	hash, _ := auth.HashPassword("pw")
	u := models.User{Username: name, PasswordHash: hash, Role: models.RoleUser}
	if err := srv.db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	srv.store.EnsureUserRoot(u.ID)
	tok, _ := srv.jwt.Issue(&u, auth.AccessToken)
	return &u, tok
}

func uploadFile(t *testing.T, srv *Server, token, destPath, filename, content string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filename)
	fw.Write([]byte(content))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload?path="+destPath, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestFSCRUDFlow(t *testing.T) {
	srv := newTestServer(t)
	tok := onboardAdmin(t, srv)

	// mkdir music
	if rec := doJSON(t, srv, http.MethodPost, "/api/fs/mkdir", `{"path":"music"}`, tok); rec.Code != http.StatusCreated {
		t.Fatalf("mkdir: %d %s", rec.Code, rec.Body)
	}

	// upload into music
	if rec := uploadFile(t, srv, tok, "music", "a.mp3", "AUDIO"); rec.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}

	// list music -> contains a.mp3
	rec := doJSON(t, srv, http.MethodGet, "/api/fs/list?path=music", "", tok)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "a.mp3") {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}

	// rename a.mp3 -> b.mp3
	if rec := doJSON(t, srv, http.MethodPost, "/api/fs/rename", `{"path":"music/a.mp3","newName":"b.mp3"}`, tok); rec.Code != http.StatusNoContent {
		t.Fatalf("rename: %d %s", rec.Code, rec.Body)
	}

	// copy b.mp3 -> music/c.mp3
	if rec := doJSON(t, srv, http.MethodPost, "/api/fs/copy", `{"src":"music/b.mp3","dst":"music/c.mp3"}`, tok); rec.Code != http.StatusNoContent {
		t.Fatalf("copy: %d %s", rec.Code, rec.Body)
	}

	// move b.mp3 -> moved/b.mp3
	if rec := doJSON(t, srv, http.MethodPost, "/api/fs/move", `{"src":"music/b.mp3","dst":"moved/b.mp3"}`, tok); rec.Code != http.StatusNoContent {
		t.Fatalf("move: %d %s", rec.Code, rec.Body)
	}

	// download c.mp3
	rec = doJSON(t, srv, http.MethodGet, "/api/download?path=music/c.mp3", "", tok)
	if rec.Code != http.StatusOK || rec.Body.String() != "AUDIO" {
		t.Fatalf("download: %d body=%q", rec.Code, rec.Body.String())
	}

	// download dir music -> zip
	rec = doJSON(t, srv, http.MethodGet, "/api/download?path=music", "", tok)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("zip download: %d ct=%s", rec.Code, rec.Header().Get("Content-Type"))
	}

	// delete music
	if rec := doJSON(t, srv, http.MethodPost, "/api/fs/delete", `{"path":"music"}`, tok); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}

	// cannot delete root
	if rec := doJSON(t, srv, http.MethodPost, "/api/fs/delete", `{"path":""}`, tok); rec.Code != http.StatusBadRequest {
		t.Fatalf("delete root should fail, got %d", rec.Code)
	}
}

func TestFSCrossUserIsolationHTTP(t *testing.T) {
	srv := newTestServer(t)
	adminTok := onboardAdmin(t, srv) // user 1
	user2, _ := makeUser(t, srv, "bob")

	// user 2 puts a secret file
	_ = user2

	// user 1 tries to traverse into user 2's root -> 400
	rec := doJSON(t, srv, http.MethodGet, "/api/fs/list?path=../"+strings.TrimPrefix(srv.store.UserRoot(user2.ID), "/"), "", adminTok)
	if rec.Code == http.StatusOK {
		t.Fatalf("cross-user list must not succeed, got %d %s", rec.Code, rec.Body)
	}

	// classic traversal payload
	rec = doJSON(t, srv, http.MethodGet, "/api/fs/list?path=../2", "", adminTok)
	if rec.Code == http.StatusOK {
		t.Fatalf("traversal must not succeed, got %d", rec.Code)
	}
}
