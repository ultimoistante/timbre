package subsonic

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/models"
)

// radioDTO converts a saved web radio to its Subsonic representation. Clients
// connect to StreamURL directly (it is the upstream URL, not proxied through
// timbre), matching how other Subsonic servers expose internet radio.
func radioDTO(st models.RadioStation) InternetRadioStation {
	return InternetRadioStation{
		ID:          RadioID(st.ID),
		Name:        st.Name,
		StreamURL:   st.URL,
		HomePageURL: st.Homepage,
	}
}

// getInternetRadioStations lists the user's saved web radios.
func (h *Handlers) getInternetRadioStations(c echo.Context) error {
	u := CurrentUser(c)
	var stations []models.RadioStation
	h.db.Where("user_id = ?", u.ID).Order("pinned DESC, name").Find(&stations)

	out := make([]InternetRadioStation, len(stations))
	for i, st := range stations {
		out[i] = radioDTO(st)
	}
	return Write(c, func(r *Response) {
		r.InternetRadioStations = &InternetRadioStations{InternetRadioStation: out}
	})
}

// createInternetRadioStation adds a new web radio station.
func (h *Handlers) createInternetRadioStation(c echo.Context) error {
	u := CurrentUser(c)

	name := param(c, "name")
	streamURL := strings.TrimSpace(param(c, "streamUrl"))
	if name == "" || streamURL == "" {
		return WriteError(c, ErrMissingPar, "Required parameter 'name' or 'streamUrl' is missing.")
	}

	st := models.RadioStation{
		UserID:   u.ID,
		Name:     name,
		URL:      streamURL,
		Homepage: param(c, "homepageUrl"),
	}
	if err := h.db.Create(&st).Error; err != nil {
		return WriteError(c, ErrGeneric, "Could not create radio station.")
	}
	return Write(c, nil)
}

// updateInternetRadioStation edits name/streamUrl/homepageUrl of a station.
func (h *Handlers) updateInternetRadioStation(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindRadio {
		return WriteError(c, ErrNotFound, "Internet radio station not found.")
	}
	var st models.RadioStation
	if err := h.db.Where("id = ? AND user_id = ?", p.UintID, u.ID).First(&st).Error; err != nil {
		return WriteError(c, ErrNotFound, "Internet radio station not found.")
	}

	if name := param(c, "name"); name != "" {
		st.Name = name
	}
	if streamURL := strings.TrimSpace(param(c, "streamUrl")); streamURL != "" {
		st.URL = streamURL
	}
	if homepage := param(c, "homepageUrl"); homepage != "" {
		st.Homepage = homepage
	}
	h.db.Save(&st)
	return Write(c, nil)
}

// deleteInternetRadioStation removes a station.
func (h *Handlers) deleteInternetRadioStation(c echo.Context) error {
	u := CurrentUser(c)
	p, err := ParseID(param(c, "id"))
	if err != nil || p.Kind != KindRadio {
		return WriteError(c, ErrNotFound, "Internet radio station not found.")
	}
	h.db.Where("id = ? AND user_id = ?", p.UintID, u.ID).Delete(&models.RadioStation{})
	return Write(c, nil)
}
