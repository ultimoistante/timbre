package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// makeTestAudio generates a 1s sine mp3 in the user's root, seeds a DB row and
// returns its MediaFile ID.
func makeTestAudio(t *testing.T, srv *Server, userID uint, relPath, title string) uint {
	t.Helper()
	root := srv.store.UserRoot(userID)
	abs := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-metadata", "title="+title, abs)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v\n%s", err, out)
	}
	seedTrack(t, srv, userID, relPath, title, "Album", "Artist", 1.0)

	var row struct{ ID uint }
	srv.db.Raw("SELECT id FROM media_files WHERE user_id=? AND rel_path=?", userID, relPath).Scan(&row)
	return row.ID
}

func uintStr(n uint) string { return strconv.FormatUint(uint64(n), 10) }

func rangeRequest(t *testing.T, srv *Server, path, bearer, rangeHdr string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+bearer)
	if rangeHdr != "" {
		req.Header.Set("Range", rangeHdr)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestStreamOriginalAndRange(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	srv := newTestServer(t)
	tok := onboardAdmin(t, srv)
	id := makeTestAudio(t, srv, 1, "music/test.mp3", "TestSong")
	url := "/api/stream/" + uintStr(id)

	// Full GET: must succeed and return audio data.
	rec := rangeRequest(t, srv, url, tok, "")
	if rec.Code != http.StatusOK && rec.Code != http.StatusPartialContent {
		t.Fatalf("stream GET: %d %s", rec.Code, rec.Body)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("empty body from stream")
	}

	// Range request: first 512 bytes → 206.
	rec = rangeRequest(t, srv, url, tok, "bytes=0-511")
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("range request: %d (want 206)", rec.Code)
	}
	if rec.Body.Len() != 512 {
		t.Fatalf("range body size: %d (want 512)", rec.Body.Len())
	}
}

func TestStreamCrossUserBlocked(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	srv := newTestServer(t)
	_ = onboardAdmin(t, srv)
	_, bobTok := makeUser(t, srv, "bob")
	id := makeTestAudio(t, srv, 1, "music/admin.mp3", "AdminSong")

	// Bob must not access admin's track.
	rec := rangeRequest(t, srv, "/api/stream/"+uintStr(id), bobTok, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-user stream should be 404, got %d", rec.Code)
	}
}

func TestStreamTranscode(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	srv := newTestServer(t)
	tok := onboardAdmin(t, srv)
	id := makeTestAudio(t, srv, 1, "music/test.mp3", "TestSong")

	rec := rangeRequest(t, srv, "/api/stream/"+uintStr(id)+"?quality=64k&container=mp3", tok, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("transcode: %d %s", rec.Code, rec.Body)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "audio/mpeg" {
		t.Fatalf("content-type: %q", ct)
	}
	if rec.Header().Get("X-Transcoded-Bitrate") != "64k" {
		t.Fatalf("X-Transcoded-Bitrate missing/wrong")
	}
	if rec.Body.Len() == 0 {
		t.Fatal("empty transcode body")
	}
}
