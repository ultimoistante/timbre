package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	usersDir := filepath.Join(dir, "users")
	if err := os.MkdirAll(usersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s := New(usersDir)
	if err := s.EnsureUserRoot(1); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureUserRoot(2); err != nil {
		t.Fatal(err)
	}
	return s, usersDir
}

func TestResolveValidPaths(t *testing.T) {
	s, usersDir := newStore(t)
	root := filepath.Join(usersDir, "1")

	cases := map[string]string{
		"":                  root,
		".":                 root,
		"/":                 root,
		"song.mp3":          filepath.Join(root, "song.mp3"),
		"/album/song.mp3":   filepath.Join(root, "album", "song.mp3"),
		"album/../song.mp3": filepath.Join(root, "song.mp3"), // internal .. that stays inside
	}
	for in, want := range cases {
		got, err := s.Resolve(1, in)
		if err != nil {
			t.Errorf("Resolve(%q) unexpected error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("Resolve(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveRejectsTraversal(t *testing.T) {
	s, _ := newStore(t)

	bad := []string{
		"../2/secret.mp3",
		"../../etc/passwd",
		"..",
		"a/b/../../../etc/passwd",
		"foo/../../bar",
	}
	for _, in := range bad {
		if got, err := s.Resolve(1, in); err == nil {
			t.Errorf("Resolve(%q) should fail, got %q", in, got)
		}
	}
}

func TestResolveRejectsAbsoluteEscape(t *testing.T) {
	s, _ := newStore(t)
	// An absolute path should be treated relative to the user root, never the
	// real filesystem root.
	got, err := s.Resolve(1, "/etc/passwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, filepath.Join("users", "1")) {
		t.Fatalf("absolute path escaped user root: %q", got)
	}
}

func TestResolveRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	s, usersDir := newStore(t)
	root := filepath.Join(usersDir, "1")
	outside := t.TempDir()

	// Create a symlink inside user 1's root pointing outside.
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	if got, err := s.Resolve(1, "escape/secret.mp3"); err == nil {
		t.Errorf("Resolve through escaping symlink should fail, got %q", got)
	}
}

func TestResolveCrossUserIsolation(t *testing.T) {
	s, _ := newStore(t)
	// User 1 must never resolve into user 2's root.
	if got, err := s.Resolve(1, "../2"); err == nil {
		t.Errorf("cross-user access should fail, got %q", got)
	}
}
