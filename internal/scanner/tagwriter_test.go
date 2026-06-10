package scanner

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ultimoistante/timbre/internal/models"
)

// genSilence creates a short silent audio file of the given extension using
// ffmpeg. Returns the path or skips the test if ffmpeg is unavailable.
func genSilence(t *testing.T, ext string) string {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	out := filepath.Join(t.TempDir(), "sample"+ext)
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono",
		"-t", "1", "-y", out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg failed: %v\n%s", err, b)
	}
	return out
}

func TestWriteTagsRoundTrip(t *testing.T) {
	for _, ext := range []string{".mp3", ".flac", ".ogg"} {
		t.Run(ext, func(t *testing.T) {
			path := genSilence(t, ext)

			want := &TagInfo{
				Title:       "Test Title",
				Album:       "Test Album",
				AlbumArtist: "Album Artist",
				Artists:     "Track Artist",
				Genres:      "Jazz",
				TrackNo:     3,
				DiscNo:      1,
				Year:        2021,
			}
			if err := WriteTags(path, want); err != nil {
				t.Fatalf("WriteTags: %v", err)
			}

			got, err := ParseTags(path)
			if err != nil {
				t.Fatalf("ParseTags: %v", err)
			}
			if got.Title != want.Title {
				t.Errorf("Title = %q, want %q", got.Title, want.Title)
			}
			if got.Album != want.Album {
				t.Errorf("Album = %q, want %q", got.Album, want.Album)
			}
			if got.AlbumArtist != want.AlbumArtist {
				t.Errorf("AlbumArtist = %q, want %q", got.AlbumArtist, want.AlbumArtist)
			}
			if got.Artists != want.Artists {
				t.Errorf("Artists = %q, want %q", got.Artists, want.Artists)
			}
			if got.TrackNo != want.TrackNo {
				t.Errorf("TrackNo = %d, want %d", got.TrackNo, want.TrackNo)
			}
			if got.Year != want.Year {
				t.Errorf("Year = %d, want %d", got.Year, want.Year)
			}
		})
	}
}

func TestApplyTagFieldsRehashes(t *testing.T) {
	m := &models.MediaFile{AlbumHash: "old"}
	t1 := &TagInfo{Artists: "A", Album: "Alb", Title: "T", AlbumArtist: "AA"}
	ApplyTagFields(m, t1)

	if m.AlbumHash != AlbumHash("AA", "Alb") {
		t.Errorf("AlbumHash not recomputed: %q", m.AlbumHash)
	}
	if m.TrackHash != TrackHash("A", "Alb", "T") {
		t.Errorf("TrackHash not recomputed: %q", m.TrackHash)
	}
	if m.ArtistHash != ArtistHash("AA") {
		t.Errorf("ArtistHash not recomputed: %q", m.ArtistHash)
	}
}
