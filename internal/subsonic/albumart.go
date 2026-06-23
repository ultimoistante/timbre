package subsonic

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dhowden/tag"
	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/storage"
)

// ErrNoArt is returned when an album has no extractable cover art.
var ErrNoArt = errors.New("no album art")

// ExtractAlbumArt returns the cover art bytes and MIME type for an album,
// serving from the on-disk cache (<dataDir>/art/<hash>) when present and
// extracting + caching from an embedded picture otherwise. It is shared by the
// native /api album-art handler and the Subsonic getCoverArt endpoint.
func ExtractAlbumArt(db *gorm.DB, store *storage.Store, dataDir string, userID uint, albumHash string) ([]byte, string, error) {
	if !hashRe.MatchString(albumHash) {
		return nil, "", ErrNoArt
	}

	cacheDir := filepath.Join(dataDir, "art")
	cachePath := filepath.Join(cacheDir, albumHash)

	if data, err := os.ReadFile(cachePath); err == nil {
		return data, http.DetectContentType(data), nil
	}

	var track models.MediaFile
	if err := db.Where("user_id = ? AND album_hash = ?", userID, albumHash).
		First(&track).Error; err != nil {
		return nil, "", ErrNoArt
	}

	absPath, err := store.Resolve(userID, track.RelPath)
	if err != nil {
		return nil, "", ErrNoArt
	}

	f, err := os.Open(absPath)
	if err != nil {
		return nil, "", ErrNoArt
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, "", ErrNoArt
	}
	pic := m.Picture()
	if pic == nil || len(pic.Data) == 0 {
		return nil, "", ErrNoArt
	}

	// Cache to disk (best-effort).
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		_ = os.WriteFile(cachePath, pic.Data, 0o644)
	}

	mime := pic.MIMEType
	if mime == "" || mime == "application/octet-stream" {
		mime = http.DetectContentType(pic.Data)
	}
	return pic.Data, mime, nil
}
