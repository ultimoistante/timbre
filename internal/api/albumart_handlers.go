package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.senan.xyz/taglib"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
)

// artHTTPClient is the outbound client used for iTunes lookups and art
// downloads. A short timeout bounds slow/hostile remotes.
var artHTTPClient = &http.Client{Timeout: 15 * time.Second}

const maxArtBytes = 12 << 20 // 12 MiB cap on downloaded artwork

// ArtCandidate is one cover-art search result surfaced to the client.
type ArtCandidate struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Thumb  string `json:"thumb"` // small preview (~100px)
	URL    string `json:"url"`   // higher-res image to embed (~600px)
}

// itunesResponse is the subset of the iTunes Search API payload we read.
type itunesResponse struct {
	Results []struct {
		CollectionName string `json:"collectionName"`
		ArtistName     string `json:"artistName"`
		ArtworkURL100  string `json:"artworkUrl100"`
	} `json:"results"`
}

// handleSearchAlbumArt queries the free iTunes Search API for candidate covers.
// The search term defaults to "<album> <albumArtist>" of the album, overridable
// via ?q=.
func (s *Server) handleSearchAlbumArt(c echo.Context) error {
	u := auth.CurrentUser(c)
	albumHash := c.Param("hash")
	if !validHash.MatchString(albumHash) {
		return c.NoContent(http.StatusNotFound)
	}

	term := strings.TrimSpace(c.QueryParam("q"))
	if term == "" {
		var track models.MediaFile
		if err := s.db.Where("user_id = ? AND album_hash = ?", u.ID, albumHash).
			First(&track).Error; err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		term = strings.TrimSpace(track.Album + " " + track.AlbumArtist)
	}
	if term == "" {
		return c.JSON(http.StatusOK, []ArtCandidate{})
	}

	q := url.Values{}
	q.Set("term", term)
	q.Set("entity", "album")
	q.Set("media", "music")
	q.Set("limit", "12")
	endpoint := "https://itunes.apple.com/search?" + q.Encode()

	req, _ := http.NewRequestWithContext(c.Request().Context(), http.MethodGet, endpoint, nil)
	resp, err := artHTTPClient.Do(req)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "lookup failed"})
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "lookup failed"})
	}

	var it itunesResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&it); err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "bad lookup response"})
	}

	out := make([]ArtCandidate, 0, len(it.Results))
	for _, r := range it.Results {
		if r.ArtworkURL100 == "" {
			continue
		}
		out = append(out, ArtCandidate{
			Title:  r.CollectionName,
			Artist: r.ArtistName,
			Thumb:  r.ArtworkURL100,
			// iTunes art URLs end in "<w>x<h>bb.jpg"; request a larger render.
			URL: strings.Replace(r.ArtworkURL100, "100x100bb", "600x600bb", 1),
		})
	}
	return c.JSON(http.StatusOK, out)
}

// handleSetAlbumArt downloads an image from the given URL and embeds it as the
// front cover into every track file of the album, then refreshes the art cache.
func (s *Server) handleSetAlbumArt(c echo.Context) error {
	u := auth.CurrentUser(c)
	albumHash := c.Param("hash")
	if !validHash.MatchString(albumHash) {
		return c.NoContent(http.StatusNotFound)
	}

	var body struct {
		URL string `json:"url"`
	}
	if err := c.Bind(&body); err != nil || strings.TrimSpace(body.URL) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url required"})
	}

	img, err := s.downloadImage(c.Request().Context(), body.URL)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}

	var tracks []models.MediaFile
	if err := s.db.Where("user_id = ? AND album_hash = ?", u.ID, albumHash).
		Find(&tracks).Error; err != nil {
		return err
	}
	if len(tracks) == 0 {
		return c.NoContent(http.StatusNotFound)
	}

	for i := range tracks {
		abs, err := s.store.Resolve(u.ID, tracks[i].RelPath)
		if err != nil {
			continue
		}
		_ = taglib.WriteImage(abs, img)
	}

	// Refresh the on-disk cache so GET .../art serves the new image. Clients
	// must cache-bust the URL (the cache is served immutable).
	cacheDir := filepath.Join(s.cfg.DataDir, "art")
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		_ = os.WriteFile(filepath.Join(cacheDir, albumHash), img, 0o644)
	}

	return c.NoContent(http.StatusNoContent)
}

// downloadImage fetches url and returns the bytes, enforcing scheme, size and
// content-type guards. Note: the URL is client-supplied, so this is a server-
// side fetch — guards (https/http only, image content-type, size cap, timeout)
// limit abuse on a self-hosted server.
func (s *Server) downloadImage(ctx context.Context, raw string) ([]byte, error) {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, echoErr("invalid url")
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	resp, err := artHTTPClient.Do(req)
	if err != nil {
		return nil, echoErr("download failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, echoErr("download failed")
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArtBytes+1))
	if err != nil {
		return nil, echoErr("download failed")
	}
	if len(data) > maxArtBytes {
		return nil, echoErr("image too large")
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	if !strings.HasPrefix(ct, "image/") {
		return nil, echoErr("not an image")
	}
	return data, nil
}

// echoErr is a tiny error helper to keep messages terse and uniform.
func echoErr(msg string) error { return &simpleErr{msg} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }
