package subsonic

import (
	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/models"
)

// ping confirms the server is reachable and the credentials are valid.
func (h *Handlers) ping(c echo.Context) error {
	return Write(c, nil)
}

// getLicense always reports a valid license (self-hosted).
func (h *Handlers) getLicense(c echo.Context) error {
	return Write(c, func(r *Response) {
		r.License = &License{Valid: true}
	})
}

// getOpenSubsonicExtensions advertises supported OpenSubsonic extensions.
func (h *Handlers) getOpenSubsonicExtensions(c echo.Context) error {
	return Write(c, func(r *Response) {
		r.OpenSubsonicExtensions = []OSExtension{
			{Name: "apiKeyAuthentication", Versions: []int{1}},
		}
	})
}

// getUser reports the authenticated user's capability roles.
func (h *Handlers) getUser(c echo.Context) error {
	u := CurrentUser(c)
	return Write(c, func(r *Response) {
		r.User = &SubsonicUser{
			Username:          u.Username,
			AdminRole:         u.IsAdmin(),
			SettingsRole:      u.IsAdmin(),
			DownloadRole:      true,
			UploadRole:        u.IsAdmin(),
			PlaylistRole:      true,
			CoverArtRole:      true,
			StreamRole:        true,
			ScrobblingEnabled: true,
			Folder:            []int{1},
		}
	})
}

// getScanStatus reports the user's library size (scanning is never reported as
// in-progress here; the native /api/scan drives indexing).
func (h *Handlers) getScanStatus(c echo.Context) error {
	u := CurrentUser(c)
	var count int64
	h.db.Model(&models.MediaFile{}).Where("user_id = ?", u.ID).Count(&count)
	return Write(c, func(r *Response) {
		r.ScanStatus = &ScanStatus{Scanning: false, Count: int(count)}
	})
}
