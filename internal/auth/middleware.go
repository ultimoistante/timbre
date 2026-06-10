package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/models"
)

// userContextKey is the Echo context key under which the authenticated user is
// stored after RequireAuth succeeds.
const userContextKey = "currentUser"

// AccessCookieName is the cookie holding the access token for browser clients.
const AccessCookieName = "access_token"

// RefreshCookieName is the cookie holding the refresh token.
const RefreshCookieName = "refresh_token"

// CurrentUser returns the authenticated user from the request context, or nil.
func CurrentUser(c echo.Context) *models.User {
	if u, ok := c.Get(userContextKey).(*models.User); ok {
		return u
	}
	return nil
}

// extractToken reads the access token from the Authorization header (preferred
// for API/mobile clients) or the access cookie (browser clients).
func extractToken(c echo.Context) string {
	if h := c.Request().Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if cookie, err := c.Cookie(AccessCookieName); err == nil {
		return cookie.Value
	}
	return ""
}

// RequireAuth is middleware that validates the access token, loads the user and
// stores it in the context. Requests without a valid token get 401.
func RequireAuth(m *Manager, gdb *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tok := extractToken(c)
			if tok == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
			}

			claims, err := m.Parse(tok)
			if err != nil || claims.Type != AccessToken {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			var user models.User
			if err := gdb.First(&user, claims.UserID).Error; err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "unknown user")
			}

			c.Set(userContextKey, &user)
			return next(c)
		}
	}
}

// RequireAdmin is middleware that rejects non-admin users with 403. Must be
// chained after RequireAuth.
func RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		u := CurrentUser(c)
		if u == nil || !u.IsAdmin() {
			return echo.NewHTTPError(http.StatusForbidden, "admin only")
		}
		return next(c)
	}
}
