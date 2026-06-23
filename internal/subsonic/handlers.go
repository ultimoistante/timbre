package subsonic

import (
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/config"
	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/storage"
)

// Handlers holds the dependencies the /rest endpoints need. It reuses the
// per-user MediaFile library and the shared stream/storage logic.
type Handlers struct {
	db      *gorm.DB
	store   *storage.Store
	dataDir string
}

// NewHandlers builds the Subsonic handler set.
func NewHandlers(db *gorm.DB, store *storage.Store, cfg *config.Config) *Handlers {
	return &Handlers{db: db, store: store, dataDir: cfg.DataDir}
}

const albumAggSelect = `album_hash, album, album_artist, artist_hash,
	MAX(year) as year, COUNT(*) as track_count, SUM(duration) as duration`

const albumAggGroup = "album_hash, album, album_artist, artist_hash"

// queryAlbums runs the derived-album GROUP BY query for a user, with an
// optional extra WHERE, an ORDER BY, and limit/offset.
func (h *Handlers) queryAlbums(userID uint, where, args, order string, limit, offset int) []albumAgg {
	q := h.db.Model(&models.MediaFile{}).
		Select(albumAggSelect).
		Where("user_id = ?", userID)
	if where != "" {
		q = q.Where(where, args)
	}
	var rows []albumAgg
	q.Group(albumAggGroup).Order(order).Limit(limit).Offset(offset).Scan(&rows)
	return rows
}

// albumByHash returns the single derived album for a hash (or false).
func (h *Handlers) albumByHash(userID uint, hash string) (albumAgg, bool) {
	var a albumAgg
	err := h.db.Model(&models.MediaFile{}).
		Select(albumAggSelect).
		Where("user_id = ? AND album_hash = ?", userID, hash).
		Group(albumAggGroup).
		Scan(&a).Error
	if err != nil || a.AlbumHash == "" {
		return albumAgg{}, false
	}
	return a, true
}

// albumTracks returns the tracks of an album in disc/track order.
func (h *Handlers) albumTracks(userID uint, hash string) []models.MediaFile {
	var tracks []models.MediaFile
	h.db.Where("user_id = ? AND album_hash = ?", userID, hash).
		Order("disc_no, track_no").Find(&tracks)
	return tracks
}

// queryArtists runs the derived-artist GROUP BY query for a user.
func (h *Handlers) queryArtists(userID uint) []artistAgg {
	var rows []artistAgg
	h.db.Model(&models.MediaFile{}).
		Select(`artist_hash, album_artist as name,
			COUNT(DISTINCT album_hash) as album_count, COUNT(*) as track_count`).
		Where("user_id = ? AND album_artist <> ''", userID).
		Group("artist_hash, album_artist").
		Order("album_artist").
		Scan(&rows)
	return rows
}

// trackByID loads one of the user's tracks by numeric id.
func (h *Handlers) trackByID(userID, id uint) (models.MediaFile, bool) {
	var mf models.MediaFile
	if err := h.db.Where("user_id = ? AND id = ?", userID, id).First(&mf).Error; err != nil {
		return models.MediaFile{}, false
	}
	return mf, true
}

// bucketArtists groups artists into alphabetical index buckets (Subsonic
// indexes). Non-letter initials fall under "#".
func bucketArtists(artists []artistAgg) []IndexID3 {
	buckets := map[string][]ArtistID3{}
	for _, a := range artists {
		key := indexLetter(a.Name)
		buckets[key] = append(buckets[key], artistID3FromAgg(a))
	}
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]IndexID3, 0, len(keys))
	for _, k := range keys {
		out = append(out, IndexID3{Name: k, Artist: buckets[k]})
	}
	return out
}

// indexLetter returns the uppercase first letter of a name, or "#".
func indexLetter(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "#"
	}
	r := []rune(strings.ToUpper(name))[0]
	if r >= 'A' && r <= 'Z' {
		return string(r)
	}
	return "#"
}

// atoiDefault parses an integer string, returning def when empty or invalid.
func atoiDefault(raw string, def int) int {
	if raw == "" {
		return def
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return n
	}
	return def
}
