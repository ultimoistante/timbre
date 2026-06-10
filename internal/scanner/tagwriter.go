package scanner

import (
	"strconv"

	"go.senan.xyz/taglib"

	"github.com/ultimoistante/timbre/internal/models"
)

// WriteTags writes the editable metadata fields of t back into the audio file at
// absPath. It writes the supported formats via TagLib (mp3/flac/ogg/m4a/…) and
// merges into existing tags (opts 0) so unrelated frames are preserved. Numeric
// fields are only written when non-zero to avoid clobbering existing values.
func WriteTags(absPath string, t *TagInfo) error {
	tags := map[string][]string{
		taglib.Title:       {t.Title},
		taglib.Album:       {t.Album},
		taglib.AlbumArtist: {t.AlbumArtist},
		taglib.Artist:      {t.Artists},
		taglib.Genre:       {t.Genres},
	}
	if t.TrackNo > 0 {
		tags[taglib.TrackNumber] = []string{strconv.Itoa(t.TrackNo)}
	}
	if t.DiscNo > 0 {
		tags[taglib.DiscNumber] = []string{strconv.Itoa(t.DiscNo)}
	}
	if t.Year > 0 {
		tags[taglib.Date] = []string{strconv.Itoa(t.Year)}
	}
	return taglib.WriteTags(absPath, tags, 0)
}

// ApplyTagFields copies the tag-derived fields of t onto m and recomputes the
// track/album/artist hashes. Shared by buildMediaFile and the metadata-edit
// handlers so hashing stays identical everywhere. It does NOT touch path,
// size, mtime or the ffprobe-derived audio properties.
func ApplyTagFields(m *models.MediaFile, t *TagInfo) {
	m.Title = t.Title
	m.Album = t.Album
	m.AlbumArtist = t.AlbumArtist
	m.Artists = t.Artists
	m.Genres = t.Genres
	m.TrackNo = t.TrackNo
	m.DiscNo = t.DiscNo
	m.Year = t.Year
	m.TrackHash = TrackHash(t.Artists, t.Album, t.Title)
	m.AlbumHash = AlbumHash(t.AlbumArtist, t.Album)
	m.ArtistHash = ArtistHash(t.AlbumArtist)
}
