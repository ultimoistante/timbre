package subsonic

import (
	"strings"
	"time"

	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/stream"
)

// albumAgg mirrors the derived album row produced by the GROUP BY query (the
// native api.AlbumAgg, re-declared to avoid an import cycle).
type albumAgg struct {
	AlbumHash   string
	Album       string
	AlbumArtist string
	ArtistHash  string
	Year        int
	TrackCount  int
	Duration    float64
}

// artistAgg mirrors the derived artist row.
type artistAgg struct {
	ArtistHash string
	Name       string
	AlbumCount int
	TrackCount int
}

// songFromMediaFile converts a library track to a Subsonic song (Child).
func songFromMediaFile(mf models.MediaFile) Child {
	return Child{
		ID:          TrackID(mf.ID),
		Parent:      AlbumID(mf.AlbumHash),
		IsDir:       false,
		Title:       mf.Title,
		Album:       mf.Album,
		Artist:      artistDisplay(mf),
		Track:       mf.TrackNo,
		Year:        mf.Year,
		Genre:       firstGenre(mf.Genres),
		CoverArt:    CoverID(mf.AlbumHash),
		Size:        mf.SizeBytes,
		ContentType: stream.MimeType(mf.Container),
		Suffix:      mf.Container,
		Duration:    int(mf.Duration + 0.5),
		BitRate:     mf.Bitrate / 1000,
		DiscNumber:  mf.DiscNo,
		AlbumID:     AlbumID(mf.AlbumHash),
		ArtistID:    ArtistID(mf.ArtistHash),
		Type:        "music",
		PlayCount:   mf.PlayCount,
		Created:     timeRFC(mf.CreatedAt),
		MediaType:   "song",
	}
}

// dirChildFromAlbum converts a derived album to a directory-style Child (used by
// legacy getAlbumList / search2 where albums are folders).
func dirChildFromAlbum(a albumAgg) Child {
	return Child{
		ID:       AlbumID(a.AlbumHash),
		Parent:   ArtistID(a.ArtistHash),
		IsDir:    true,
		Title:    a.Album,
		Album:    a.Album,
		Artist:   a.AlbumArtist,
		CoverArt: CoverID(a.AlbumHash),
		Year:     a.Year,
		AlbumID:  AlbumID(a.AlbumHash),
		ArtistID: ArtistID(a.ArtistHash),
	}
}

// albumID3FromAgg converts a derived album to an ID3 album.
func albumID3FromAgg(a albumAgg) AlbumID3 {
	return AlbumID3{
		ID:        AlbumID(a.AlbumHash),
		Name:      a.Album,
		Artist:    a.AlbumArtist,
		ArtistID:  ArtistID(a.ArtistHash),
		CoverArt:  CoverID(a.AlbumHash),
		SongCount: a.TrackCount,
		Duration:  int(a.Duration + 0.5),
		Year:      a.Year,
	}
}

// artistID3FromAgg converts a derived artist to an ID3 artist.
func artistID3FromAgg(a artistAgg) ArtistID3 {
	return ArtistID3{
		ID:         ArtistID(a.ArtistHash),
		Name:       a.Name,
		AlbumCount: a.AlbumCount,
	}
}

// artistDisplay prefers the track's artist display string, falling back to the
// album artist.
func artistDisplay(mf models.MediaFile) string {
	if mf.Artists != "" {
		return mf.Artists
	}
	return mf.AlbumArtist
}

// genreSeparators are the delimiters that may join multiple genres in a tag.
const genreSeparators = ";/,|"

// firstGenre returns the first genre from a possibly multi-valued tag string.
func firstGenre(s string) string {
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, genreSeparators); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// splitGenres splits a multi-valued genre tag into trimmed, non-empty values.
func splitGenres(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return strings.ContainsRune(genreSeparators, r)
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// timeRFC formats a timestamp as Subsonic expects (ISO-8601 / RFC3339, UTC).
func timeRFC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
