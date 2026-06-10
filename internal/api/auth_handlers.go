package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/models"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// userMediaRoot returns the isolated media root for a user id and ensures it
// exists.
func (s *Server) ensureUserRoot(userID uint) error {
	root := filepath.Join(s.cfg.UsersDir(), strconv.FormatUint(uint64(userID), 10))
	return os.MkdirAll(root, 0o755)
}

// setAuthCookies writes access and refresh tokens as HttpOnly cookies for
// browser clients.
func (s *Server) setAuthCookies(c echo.Context, access, refresh string) {
	c.SetCookie(&http.Cookie{
		Name:     auth.AccessCookieName,
		Value:    access,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(s.jwt.AccessTTL()),
	})
	c.SetCookie(&http.Cookie{
		Name:     auth.RefreshCookieName,
		Value:    refresh,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(s.jwt.RefreshTTL()),
	})
}

func (s *Server) issueAndSet(c echo.Context, u *models.User) (map[string]any, error) {
	access, err := s.jwt.Issue(u, auth.AccessToken)
	if err != nil {
		return nil, err
	}
	refresh, err := s.jwt.Issue(u, auth.RefreshToken)
	if err != nil {
		return nil, err
	}
	s.setAuthCookies(c, access, refresh)
	return map[string]any{
		"accessToken":  access,
		"refreshToken": refresh,
		"user":         u,
	}, nil
}

// handleOnboardingStatus reports whether an admin already exists, so the client
// can decide whether to show the onboarding screen.
func (s *Server) handleOnboardingStatus(c echo.Context) error {
	var count int64
	s.db.Model(&models.User{}).Count(&count)
	return c.JSON(http.StatusOK, map[string]any{
		"needsOnboarding": count == 0,
	})
}

// handleOnboarding creates the first user as admin. Only allowed when no users
// exist yet.
func (s *Server) handleOnboarding(c echo.Context) error {
	var count int64
	s.db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return echo.NewHTTPError(http.StatusConflict, "onboarding already completed")
	}

	var creds credentials
	if err := c.Bind(&creds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}
	if creds.Username == "" || creds.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username and password required")
	}

	hash, err := auth.HashPassword(creds.Password)
	if err != nil {
		return err
	}

	user := models.User{
		Username:     creds.Username,
		PasswordHash: hash,
		Role:         models.RoleAdmin,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusConflict, "could not create user")
	}
	if err := s.ensureUserRoot(user.ID); err != nil {
		return err
	}

	resp, err := s.issueAndSet(c, &user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, resp)
}

// handleLogin authenticates a user and issues tokens.
func (s *Server) handleLogin(c echo.Context) error {
	var creds credentials
	if err := c.Bind(&creds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	var user models.User
	if err := s.db.Where("username = ?", creds.Username).First(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}
	if !auth.CheckPassword(user.PasswordHash, creds.Password) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	resp, err := s.issueAndSet(c, &user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// handleRefresh issues a new access token from a valid refresh token (read from
// the Authorization header or refresh cookie).
func (s *Server) handleRefresh(c echo.Context) error {
	tok := ""
	if cookie, err := c.Cookie(auth.RefreshCookieName); err == nil {
		tok = cookie.Value
	}
	if body := new(struct {
		RefreshToken string `json:"refreshToken"`
	}); c.Bind(body) == nil && body.RefreshToken != "" {
		tok = body.RefreshToken
	}
	if tok == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing refresh token")
	}

	claims, err := s.jwt.Parse(tok)
	if err != nil || claims.Type != auth.RefreshToken {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
	}

	var user models.User
	if err := s.db.First(&user, claims.UserID).Error; err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unknown user")
	}

	resp, err := s.issueAndSet(c, &user)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// handleLogout clears the auth cookies.
func (s *Server) handleLogout(c echo.Context) error {
	for _, name := range []string{auth.AccessCookieName, auth.RefreshCookieName} {
		c.SetCookie(&http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// handleMe returns the authenticated user.
func (s *Server) handleMe(c echo.Context) error {
	return c.JSON(http.StatusOK, auth.CurrentUser(c))
}
