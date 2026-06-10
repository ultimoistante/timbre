package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/ultimoistante/timbre/internal/auth"
	"github.com/ultimoistante/timbre/internal/scanner"
)

// startScanAsync launches a background scan for userID.
// Returns false if a scan is already running (no-op in that case).
func (s *Server) startScanAsync(userID uint) bool {
	if _, busy := s.scanning.LoadOrStore(userID, true); busy {
		return false
	}
	go func() {
		defer s.scanning.Delete(userID)
		_, err := s.scanner.Scan(userID, func(p scanner.Progress) {
			if b, e := json.Marshal(p); e == nil {
				s.hub.Publish(userID, "scan", b)
			}
		})
		if err != nil {
			s.hub.Publish(userID, "scan", []byte(fmt.Sprintf(`{"finished":true,"error":%q}`, err.Error())))
		}
	}()
	return true
}

// handleScan starts a background library scan for the current user and streams
// progress over SSE (event name "scan"). Returns 202 immediately. A second scan
// for the same user while one is running is rejected with 409.
func (s *Server) handleScan(c echo.Context) error {
	u := auth.CurrentUser(c)
	if !s.startScanAsync(u.ID) {
		return echo.NewHTTPError(http.StatusConflict, "scan already in progress")
	}
	return c.JSON(http.StatusAccepted, map[string]any{"started": true})
}

// handleSSE streams server-sent events for the current user (scan progress,
// library updates). Stays open until the client disconnects.
func (s *Server) handleSSE(c echo.Context) error {
	u := auth.CurrentUser(c)

	w := c.Response()
	w.Header().Set(echo.HeaderContentType, "text/event-stream")
	w.Header().Set(echo.HeaderCacheControl, "no-cache")
	w.Header().Set(echo.HeaderConnection, "keep-alive")
	w.WriteHeader(http.StatusOK)
	w.Flush()

	ch := s.hub.Subscribe(u.ID)
	defer s.hub.Unsubscribe(u.ID, ch)

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Event, msg.Data)
			w.Flush()
		}
	}
}
