package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
	"github.com/ultimoistante/timbre/internal/stream"
)

// validStreamURL reports whether raw is a well-formed http(s) URL.
func validStreamURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// handleProbeStream inspects a stream URL and returns detected metadata
// (station name, genre, homepage, logo) to auto-fill the add form. Best-effort:
// missing fields come back empty.
func (s *Server) handleProbeStream(c echo.Context) error {
	raw := c.QueryParam("url")
	if !validStreamURL(raw) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "valid http(s) url required"})
	}
	info, err := stream.ProbeStation(c.Request().Context(), strings.TrimSpace(raw))
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "could not probe stream"})
	}
	return c.JSON(http.StatusOK, info)
}

func (s *Server) handleListStreams(c echo.Context) error {
	u := auth.CurrentUser(c)
	var stations []models.RadioStation
	if err := s.db.Where("user_id = ?", u.ID).Order("pinned DESC, name").Find(&stations).Error; err != nil {
		return err
	}
	if stations == nil {
		stations = []models.RadioStation{}
	}
	return c.JSON(http.StatusOK, stations)
}

func (s *Server) handleCreateStream(c echo.Context) error {
	u := auth.CurrentUser(c)
	var body struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Genre    string `json:"genre"`
		Homepage string `json:"homepage"`
		Favicon  string `json:"favicon"`
	}
	if err := c.Bind(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name required"})
	}
	if !validStreamURL(body.URL) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "valid http(s) url required"})
	}
	st := models.RadioStation{
		UserID:   u.ID,
		Name:     body.Name,
		URL:      strings.TrimSpace(body.URL),
		Genre:    body.Genre,
		Homepage: body.Homepage,
		Favicon:  body.Favicon,
	}
	if err := s.db.Create(&st).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, st)
}

func (s *Server) handleUpdateStream(c echo.Context) error {
	u := auth.CurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))

	var st models.RadioStation
	if err := s.db.Where("id = ? AND user_id = ?", id, u.ID).First(&st).Error; err != nil {
		return c.NoContent(http.StatusNotFound)
	}

	var body struct {
		Name     *string `json:"name"`
		URL      *string `json:"url"`
		Genre    *string `json:"genre"`
		Homepage *string `json:"homepage"`
		Favicon  *string `json:"favicon"`
		Pinned   *bool   `json:"pinned"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}
	if body.Name != nil {
		if strings.TrimSpace(*body.Name) == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "name cannot be empty"})
		}
		st.Name = *body.Name
	}
	if body.URL != nil {
		if !validStreamURL(*body.URL) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "valid http(s) url required"})
		}
		st.URL = strings.TrimSpace(*body.URL)
	}
	if body.Genre != nil {
		st.Genre = *body.Genre
	}
	if body.Homepage != nil {
		st.Homepage = *body.Homepage
	}
	if body.Favicon != nil {
		st.Favicon = *body.Favicon
	}
	if body.Pinned != nil {
		st.Pinned = *body.Pinned
	}
	s.db.Save(&st)
	return c.JSON(http.StatusOK, st)
}

func (s *Server) handleDeleteStream(c echo.Context) error {
	u := auth.CurrentUser(c)
	id, _ := strconv.Atoi(c.Param("id"))
	s.db.Where("id = ? AND user_id = ?", id, u.ID).Delete(&models.RadioStation{})
	return c.NoContent(http.StatusNoContent)
}

// handleRadioPlay proxies the station's upstream audio to the client and pushes
// live ICY now-playing titles over SSE (event "nowplaying"). The <audio>
// element reading this endpoint authenticates via the access_token cookie, same
// as /stream/:id.
func (s *Server) handleRadioPlay(c echo.Context) error {
	u := auth.CurrentUser(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var st models.RadioStation
	if err := s.db.Where("id = ? AND user_id = ?", id, u.ID).First(&st).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "station not found")
	}

	onTitle := func(title string) {
		payload, _ := json.Marshal(map[string]any{"stationId": st.ID, "title": title})
		s.hub.Publish(u.ID, "nowplaying", payload)
	}

	// transcode=mp3 re-encodes the stream to MP3 for browsers that can't decode
	// the source codec (raw AAC/AAC+ live streams). Used as a client fallback.
	transcode := c.QueryParam("transcode") == "mp3"

	ctx := c.Request().Context()
	if err := stream.ProxyRadio(ctx, c.Response().Writer, st.URL, transcode, onTitle); err != nil {
		// Connection may already be partly written; log via Echo and return.
		c.Logger().Warnf("radio proxy %d: %v", st.ID, err)
	}
	return nil
}
