package scanner

import (
	"io/fs"
	"os"
	"path/filepath"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/storage"
)

// Scanner indexes a user's media root into the MediaFile table.
type Scanner struct {
	db    *gorm.DB
	store *storage.Store
}

// New creates a Scanner.
func New(db *gorm.DB, store *storage.Store) *Scanner {
	return &Scanner{db: db, store: store}
}

// Progress reports scan state for SSE consumers.
type Progress struct {
	Total    int    `json:"total"`
	Done     int    `json:"done"`
	Current  string `json:"current"`
	Added    int    `json:"added"`
	Updated  int    `json:"updated"`
	Removed  int    `json:"removed"`
	Finished bool   `json:"finished"`
	Error    string `json:"error,omitempty"`
}

// Result summarises a completed scan.
type Result struct {
	Added, Updated, Removed int
}

// Scan walks the user's media root and reconciles the MediaFile table. The
// optional report callback is invoked as files are processed (and once at the
// end with Finished=true). It is incremental: unchanged files (same mtime) are
// skipped, and DB rows for files no longer on disk are removed.
func (s *Scanner) Scan(userID uint, report func(Progress)) (Result, error) {
	root := s.store.UserRoot(userID)
	var res Result

	emit := func(p Progress) {
		if report != nil {
			report(p)
		}
	}

	// Existing rows: relPath -> (id, modTime).
	type row struct {
		ID      uint
		ModTime int64
	}
	existing := map[string]row{}
	var rows []models.MediaFile
	if err := s.db.Where("user_id = ?", userID).
		Select("id", "rel_path", "mod_time").Find(&rows).Error; err != nil {
		return res, err
	}
	for _, r := range rows {
		existing[r.RelPath] = row{ID: r.ID, ModTime: r.ModTime}
	}

	// Collect audio files on disk.
	type disktrack struct {
		rel     string
		abs     string
		modTime int64
		size    int64
	}
	var found []disktrack
	seen := map[string]bool{}

	walkErr := filepath.WalkDir(root, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() || !IsAudio(abs) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		seen[rel] = true
		found = append(found, disktrack{rel: rel, abs: abs, modTime: info.ModTime().Unix(), size: info.Size()})
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		return res, walkErr
	}

	total := len(found)
	for i, dt := range found {
		emit(Progress{Total: total, Done: i, Current: dt.rel, Added: res.Added, Updated: res.Updated})

		if prev, ok := existing[dt.rel]; ok && prev.ModTime == dt.modTime {
			continue // unchanged
		}

		mf := s.buildMediaFile(userID, dt.rel, dt.abs, dt.modTime, dt.size)
		if prev, ok := existing[dt.rel]; ok {
			mf.ID = prev.ID
			res.Updated++
		} else {
			res.Added++
		}

		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "rel_path"}},
			UpdateAll: true,
		}).Create(&mf).Error; err != nil {
			return res, err
		}
	}

	// Remove rows for files no longer present.
	var stale []uint
	for rel, r := range existing {
		if !seen[rel] {
			stale = append(stale, r.ID)
		}
	}
	if len(stale) > 0 {
		if err := s.db.Where("user_id = ? AND id IN ?", userID, stale).
			Delete(&models.MediaFile{}).Error; err != nil {
			return res, err
		}
		res.Removed = len(stale)
	}

	emit(Progress{Total: total, Done: total, Added: res.Added, Updated: res.Updated, Removed: res.Removed, Finished: true})
	return res, nil
}

func (s *Scanner) buildMediaFile(userID uint, rel, abs string, modTime, size int64) models.MediaFile {
	t, _ := ParseTags(abs)
	mf := models.MediaFile{
		UserID:     userID,
		RelPath:    rel,
		Duration:   t.Duration,
		Bitrate:    t.Bitrate,
		SampleRate: t.SampleRate,
		Container:  t.Container,
		SizeBytes:  size,
		ModTime:    modTime,
	}
	ApplyTagFields(&mf, t)
	return mf
}
