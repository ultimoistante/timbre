package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/scanner"
)

// trackTagPatch is the partial set of editable tag fields. Pointer fields let a
// client send only what changes; nil means "leave as-is".
type trackTagPatch struct {
	Title       *string `json:"title"`
	Album       *string `json:"album"`
	AlbumArtist *string `json:"albumArtist"`
	Artists     *string `json:"artists"`
	Genres      *string `json:"genres"`
	Year        *int    `json:"year"`
	TrackNo     *int    `json:"trackNo"`
	DiscNo      *int    `json:"discNo"`
}

// tagInfoFromRecord seeds a TagInfo from the current MediaFile values.
func tagInfoFromRecord(m *models.MediaFile) *scanner.TagInfo {
	return &scanner.TagInfo{
		Title:       m.Title,
		Album:       m.Album,
		AlbumArtist: m.AlbumArtist,
		Artists:     m.Artists,
		Genres:      m.Genres,
		TrackNo:     m.TrackNo,
		DiscNo:      m.DiscNo,
		Year:        m.Year,
	}
}

func (p trackTagPatch) applyTo(t *scanner.TagInfo) {
	if p.Title != nil {
		t.Title = *p.Title
	}
	if p.Album != nil {
		t.Album = *p.Album
	}
	if p.AlbumArtist != nil {
		t.AlbumArtist = *p.AlbumArtist
	}
	if p.Artists != nil {
		t.Artists = *p.Artists
	}
	if p.Genres != nil {
		t.Genres = *p.Genres
	}
	if p.Year != nil {
		t.Year = *p.Year
	}
	if p.TrackNo != nil {
		t.TrackNo = *p.TrackNo
	}
	if p.DiscNo != nil {
		t.DiscNo = *p.DiscNo
	}
}

// writeAndSync writes tags into the file on disk, then updates the DB record:
// re-derives the tag fields + hashes and bumps ModTime to the file's new mtime
// so the next incremental scan does not re-parse and clobber the edit.
func (s *Server) writeAndSync(tx *gorm.DB, userID uint, m *models.MediaFile, t *scanner.TagInfo) error {
	abs, err := s.store.Resolve(userID, m.RelPath)
	if err != nil {
		return err
	}
	if err := scanner.WriteTags(abs, t); err != nil {
		return err
	}
	if fi, err := os.Stat(abs); err == nil {
		m.ModTime = fi.ModTime().Unix()
	}
	scanner.ApplyTagFields(m, t)
	return tx.Save(m).Error
}

// dropArtCache best-effort removes a now-orphaned cached album-art file.
func (s *Server) dropArtCache(albumHash string) {
	if albumHash == "" {
		return
	}
	_ = os.Remove(filepath.Join(s.cfg.DataDir, "art", albumHash))
}

// handleUpdateTrack edits one track's tags, writes them to the file and syncs DB.
func (s *Server) handleUpdateTrack(c echo.Context) error {
	u := auth.CurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var m models.MediaFile
	if err := s.db.Where("id = ? AND user_id = ?", id, u.ID).First(&m).Error; err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	var patch trackTagPatch
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	oldAlbumHash := m.AlbumHash
	t := tagInfoFromRecord(&m)
	patch.applyTo(t)

	if err := s.writeAndSync(s.db, u.ID, &m, t); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if m.AlbumHash != oldAlbumHash {
		s.dropArtCache(oldAlbumHash)
	}
	return c.JSON(http.StatusOK, m)
}

// albumTagPatch is the album-level editable set, applied to every track of the album.
type albumTagPatch struct {
	Album       *string `json:"album"`
	AlbumArtist *string `json:"albumArtist"`
	Genres      *string `json:"genres"`
	Year        *int    `json:"year"`
}

// handleUpdateAlbum edits album-level tags across all tracks of an album.
// Because album/albumArtist feed the album hash, the album's hash can change;
// the new hash is returned so the client can navigate to it.
func (s *Server) handleUpdateAlbum(c echo.Context) error {
	u := auth.CurrentUser(c)
	oldHash := c.Param("hash")

	var tracks []models.MediaFile
	if err := s.db.Where("user_id = ? AND album_hash = ?", u.ID, oldHash).
		Order("disc_no, track_no").Find(&tracks).Error; err != nil {
		return err
	}
	if len(tracks) == 0 {
		return c.NoContent(http.StatusNotFound)
	}

	var patch albumTagPatch
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for i := range tracks {
			t := tagInfoFromRecord(&tracks[i])
			if patch.Album != nil {
				t.Album = *patch.Album
			}
			if patch.AlbumArtist != nil {
				t.AlbumArtist = *patch.AlbumArtist
			}
			if patch.Genres != nil {
				t.Genres = *patch.Genres
			}
			if patch.Year != nil {
				t.Year = *patch.Year
			}
			if err := s.writeAndSync(tx, u.ID, &tracks[i], t); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	newHash := tracks[0].AlbumHash
	if newHash != oldHash {
		s.dropArtCache(oldHash)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"albumHash": newHash,
		"tracks":    tracks,
	})
}
