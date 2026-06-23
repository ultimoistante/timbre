package subsonic

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/ultimoistante/timbre/internal/models"
)

const userCtxKey = "subsonicUser"

// CurrentUser returns the authenticated Subsonic user from the context.
func CurrentUser(c echo.Context) *models.User {
	u, _ := c.Get(userCtxKey).(*models.User)
	return u
}

// param reads a Subsonic parameter from the query string, falling back to the
// form body (some clients POST form-encoded requests).
func param(c echo.Context, name string) string {
	if v := c.QueryParam(name); v != "" {
		return v
	}
	return c.FormValue(name)
}

// Authenticate resolves the Subsonic credentials to a timbre user and stores it
// in the context. It supports OpenSubsonic apiKey auth, legacy token+salt auth
// and legacy password auth — all backed by the user's SubsonicToken.
func Authenticate(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := resolveUser(c, db)
			if !ok {
				return WriteError(c, ErrWrongAuth, "Wrong username or password.")
			}
			c.Set(userCtxKey, user)
			return next(c)
		}
	}
}

func resolveUser(c echo.Context, db *gorm.DB) (*models.User, bool) {
	// 1. OpenSubsonic apiKey auth (u may be omitted).
	if key := param(c, "apiKey"); key != "" {
		var u models.User
		if err := db.Where("subsonic_token = ?", key).First(&u).Error; err == nil && u.SubsonicToken != "" {
			return &u, true
		}
		return nil, false
	}

	// All remaining schemes require a username with a token set.
	username := param(c, "u")
	if username == "" {
		return nil, false
	}
	var u models.User
	if err := db.Where("username = ?", username).First(&u).Error; err != nil || u.SubsonicToken == "" {
		return nil, false
	}

	// 2. Token + salt: t = md5(token + salt).
	if t, s := param(c, "t"), param(c, "s"); t != "" && s != "" {
		sum := md5.Sum([]byte(u.SubsonicToken + s))
		want := hex.EncodeToString(sum[:])
		if subtle.ConstantTimeCompare([]byte(strings.ToLower(t)), []byte(want)) == 1 {
			return &u, true
		}
		return nil, false
	}

	// 3. Password: p = token, or p = enc:<hex(token)>.
	if p := param(c, "p"); p != "" {
		if enc, ok := strings.CutPrefix(p, "enc:"); ok {
			if dec, err := hex.DecodeString(enc); err == nil {
				p = string(dec)
			}
		}
		if subtle.ConstantTimeCompare([]byte(p), []byte(u.SubsonicToken)) == 1 {
			return &u, true
		}
	}
	return nil, false
}
