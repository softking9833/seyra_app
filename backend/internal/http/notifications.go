package httpapi

import (
	"net/http"

	"seyra/backend/internal/notify"
)

type registerDeviceRequest struct {
	Platform string `json:"platform"`
	Token    string `json:"token"`
}

type notificationPreferencesRequest struct {
	MessagesEnabled bool `json:"messages_enabled"`
	CallsEnabled    bool `json:"calls_enabled"`
	ShowPreview     bool `json:"show_preview"`
}

func (s *Server) registerDevice(w http.ResponseWriter, r *http.Request) {
	if s.alerts == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Notifications are not configured")
		return
	}
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req registerDeviceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	device, err := s.alerts.RegisterDevice(r.Context(), actorID, req.Platform, req.Token)
	if err != nil {
		writeNotifyError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       device.ID,
		"platform": device.Platform,
	})
}

func (s *Server) unregisterDevice(w http.ResponseWriter, r *http.Request) {
	if s.alerts == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Notifications are not configured")
		return
	}
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.alerts.UnregisterDevice(r.Context(), actorID, r.PathValue("device_id")); err != nil {
		writeNotifyError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	if s.alerts == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Notifications are not configured")
		return
	}
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	prefs, err := s.alerts.Preferences(r.Context(), actorID)
	if err != nil {
		writeNotifyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, notificationPreferencesJSON(prefs))
}

func (s *Server) putNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	if s.alerts == nil {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Notifications are not configured")
		return
	}
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req notificationPreferencesRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	prefs, err := s.alerts.UpdatePreferences(
		r.Context(),
		actorID,
		req.MessagesEnabled,
		req.CallsEnabled,
		req.ShowPreview,
	)
	if err != nil {
		writeNotifyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, notificationPreferencesJSON(prefs))
}

func notificationPreferencesJSON(prefs notify.Preferences) map[string]any {
	return map[string]any{
		"messages_enabled": prefs.MessagesEnabled,
		"calls_enabled":    prefs.CallsEnabled,
		"show_preview":     prefs.ShowPreview,
	}
}

func writeNotifyError(w http.ResponseWriter, err error) {
	switch err {
	case notify.ErrInvalidInput:
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid request")
	case notify.ErrNotFound:
		writeError(w, http.StatusNotFound, "not_found", "Not found")
	case notify.ErrUnauthorized:
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authorized")
	default:
		writeError(w, http.StatusInternalServerError, "internal", "Request failed")
	}
}
