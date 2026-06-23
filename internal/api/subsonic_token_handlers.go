package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
)

// minCustomTokenLen is the shortest user-chosen Subsonic password accepted.
const minCustomTokenLen = 8

// tokenConnectionHint bundles everything a user needs to configure a Subsonic
// client. The token is stored plaintext on purpose (a revocable secret, not the
// account password) so the Subsonic token-auth scheme can verify it.
type subsonicTokenResponse struct {
	Username string `json:"username"`
	Token    string `json:"token"`
	// RestURL is the base path Subsonic clients append "/<method>" to.
	RestURL string `json:"restUrl"`
}

func (s *Server) subsonicTokenResponse(c echo.Context, u *models.User) subsonicTokenResponse {
	scheme := "http"
	if c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return subsonicTokenResponse{
		Username: u.Username,
		Token:    u.SubsonicToken,
		RestURL:  scheme + "://" + c.Request().Host + "/rest",
	}
}

// handleGetSubsonicToken returns the current Subsonic token, or 404 if unset.
func (s *Server) handleGetSubsonicToken(c echo.Context) error {
	u := auth.CurrentUser(c)
	if u.SubsonicToken == "" {
		return c.NoContent(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, s.subsonicTokenResponse(c, u))
}

// handleRotateSubsonicToken generates (or replaces) the user's Subsonic token.
func (s *Server) handleRotateSubsonicToken(c echo.Context) error {
	u := auth.CurrentUser(c)

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	u.SubsonicToken = hex.EncodeToString(buf)
	if err := s.db.Model(u).Update("subsonic_token", u.SubsonicToken).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, s.subsonicTokenResponse(c, u))
}

// handleSetSubsonicToken sets a user-chosen Subsonic token (a custom, memorable
// password that is easier to type on a mobile client than the random default).
func (s *Server) handleSetSubsonicToken(c echo.Context) error {
	u := auth.CurrentUser(c)

	var body struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	token := strings.TrimSpace(body.Token)
	if len(token) < minCustomTokenLen {
		return echo.NewHTTPError(http.StatusBadRequest,
			"token must be at least 8 characters")
	}

	u.SubsonicToken = token
	if err := s.db.Model(u).Update("subsonic_token", token).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, s.subsonicTokenResponse(c, u))
}

// handleRevokeSubsonicToken clears the user's Subsonic token, disabling /rest
// access for them.
func (s *Server) handleRevokeSubsonicToken(c echo.Context) error {
	u := auth.CurrentUser(c)
	if err := s.db.Model(u).Update("subsonic_token", "").Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
