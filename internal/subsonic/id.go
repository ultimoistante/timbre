package subsonic

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// hashRe matches the 16-char hex album/artist hashes used by timbre (same
// pattern as the native album-art handler).
var hashRe = regexp.MustCompile(`^[0-9a-f]{16}$`)

// Subsonic IDs are opaque strings. Timbre prefixes them by entity kind so the
// /rest layer can route an id back to the right native lookup.
func TrackID(id uint) string      { return "tr-" + strconv.FormatUint(uint64(id), 10) }
func AlbumID(hash string) string  { return "al-" + hash }
func ArtistID(hash string) string { return "ar-" + hash }
func PlaylistID(id uint) string   { return "pl-" + strconv.FormatUint(uint64(id), 10) }
func CoverID(hash string) string  { return "co-" + hash }

// Kind classifies a parsed Subsonic id.
type Kind int

const (
	KindUnknown Kind = iota
	KindTrack
	KindAlbum
	KindArtist
	KindPlaylist
	KindCover
)

// ParsedID is the decoded form of an opaque Subsonic id.
type ParsedID struct {
	Kind   Kind
	UintID uint   // set for track/playlist
	Hash   string // set for album/artist/cover
}

// ParseID decodes an opaque Subsonic id by its prefix. A bare integer is
// tolerated as a track id (some clients round-trip raw numeric ids).
func ParseID(s string) (ParsedID, error) {
	if s == "" {
		return ParsedID{}, errors.New("empty id")
	}
	prefix, rest, ok := strings.Cut(s, "-")
	if !ok {
		if n, err := strconv.ParseUint(s, 10, 64); err == nil {
			return ParsedID{Kind: KindTrack, UintID: uint(n)}, nil
		}
		return ParsedID{}, errors.New("invalid id")
	}
	switch prefix {
	case "tr":
		n, err := strconv.ParseUint(rest, 10, 64)
		if err != nil {
			return ParsedID{}, err
		}
		return ParsedID{Kind: KindTrack, UintID: uint(n)}, nil
	case "pl":
		n, err := strconv.ParseUint(rest, 10, 64)
		if err != nil {
			return ParsedID{}, err
		}
		return ParsedID{Kind: KindPlaylist, UintID: uint(n)}, nil
	case "al":
		if !hashRe.MatchString(rest) {
			return ParsedID{}, errors.New("bad album hash")
		}
		return ParsedID{Kind: KindAlbum, Hash: rest}, nil
	case "ar":
		if !hashRe.MatchString(rest) {
			return ParsedID{}, errors.New("bad artist hash")
		}
		return ParsedID{Kind: KindArtist, Hash: rest}, nil
	case "co":
		if !hashRe.MatchString(rest) {
			return ParsedID{}, errors.New("bad cover hash")
		}
		return ParsedID{Kind: KindCover, Hash: rest}, nil
	}
	return ParsedID{}, errors.New("unknown id prefix")
}
