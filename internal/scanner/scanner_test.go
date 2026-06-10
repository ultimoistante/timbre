package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/storage"
)

func newScanner(t *testing.T) (*Scanner, *storage.Store, *gorm.DB, string) {
	t.Helper()
	dir := t.TempDir()
	usersDir := filepath.Join(dir, "users")
	st := storage.New(usersDir)
	if err := st.EnsureUserRoot(1); err != nil {
		t.Fatal(err)
	}

	gdb, err := gorm.Open(sqlite.Open(filepath.Join(dir, "t.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatal(err)
	}
	return New(gdb, st), st, gdb, usersDir
}

// makeAudio generates a 1s sine mp3 with metadata via ffmpeg.
func makeAudio(t *testing.T, path, title, album, artist string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-metadata", "title="+title, "-metadata", "album="+album, "-metadata", "artist="+artist,
		path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v\n%s", err, out)
	}
}

func TestScanIncremental(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	sc, st, gdb, _ := newScanner(t)
	root := st.UserRoot(1)

	makeAudio(t, filepath.Join(root, "Artist", "Album", "01.mp3"), "Song One", "Greatest", "Artist")
	makeAudio(t, filepath.Join(root, "Artist", "Album", "02.mp3"), "Song Two", "Greatest", "Artist")

	// First scan: 2 added.
	res, err := sc.Scan(1, nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Added != 2 {
		t.Fatalf("expected 2 added, got %+v", res)
	}

	var count int64
	gdb.Model(&models.MediaFile{}).Where("user_id = ?", 1).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}

	// Verify metadata + duration (ffprobe) captured.
	var mf models.MediaFile
	gdb.Where("title = ?", "Song One").First(&mf)
	if mf.Album != "Greatest" || mf.Duration <= 0 {
		t.Fatalf("metadata not captured: %+v", mf)
	}

	// Re-scan unchanged: nothing added/updated.
	res, _ = sc.Scan(1, nil)
	if res.Added != 0 || res.Updated != 0 {
		t.Fatalf("unchanged rescan should be no-op, got %+v", res)
	}

	// Delete a file on disk, re-scan removes its row.
	os.Remove(filepath.Join(root, "Artist", "Album", "02.mp3"))
	res, _ = sc.Scan(1, nil)
	if res.Removed != 1 {
		t.Fatalf("expected 1 removed, got %+v", res)
	}
	gdb.Model(&models.MediaFile{}).Where("user_id = ?", 1).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 row after delete, got %d", count)
	}
}
