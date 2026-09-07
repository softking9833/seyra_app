package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"seyra/backend/internal/auth"
)

type Server struct {
	auth *auth.Service
}

func NewServer(authService *auth.Service) http.Handler {
	s := &Server{auth: authService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /v1/auth/register", s.register)
	mux.HandleFunc("POST /v1/auth/login", s.login)
	mux.HandleFunc("GET /v1/auth/session", s.session)
	mux.HandleFunc("POST /v1/auth/logout", s.logout)
	mux.HandleFunc("POST /v1/auth/refresh", s.refresh)
	return withLogging(withMaxBody(mux))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type passwordRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	req, ok := decodePassword(w, r)
	if !ok {
		return
	}
	issued, err := s.auth.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sessionPayload(issued))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	req, ok := decodePassword(w, r)
	if !ok {
		return
	}
	issued, err := s.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionPayload(issued))
}

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	user, session, err := s.auth.CurrentSession(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
		},
		"session": map[string]string{
			"id":         session.ID,
			"expires_at": session.ExpiresAt.UTC().Format(time.RFC3339Nano),
		},
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if err := s.auth.Logout(r.Context(), token); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	issued, err := s.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionPayload(issued))
}

func decodePassword(w http.ResponseWriter, r *http.Request) (passwordRequest, bool) {
	var req passwordRequest
	if !decodeJSON(w, r, &req) {
		return passwordRequest{}, false
	}
	return req, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid request body")
		return false
	}
	return true
}

func sessionPayload(issued auth.IssuedSession) map[string]any {
	return map[string]any{
		"user": map[string]string{
			"id":       issued.User.ID,
			"username": issued.User.Username,
		},
		"session": map[string]string{
			"id":         issued.Session.ID,
			"expires_at": issued.Session.ExpiresAt.UTC().Format(time.RFC3339Nano),
		},
		"credentials": map[string]any{
			"access_token":  issued.AccessToken,
			"refresh_token": issued.RefreshToken,
			"token_type":    "Bearer",
			"expires_in":    issued.ExpiresIn,
		},
	}
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, auth.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "username_taken", "Username is already taken")
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
	case errors.Is(err, auth.ErrSessionExpired):
		writeError(w, http.StatusUnauthorized, "session_expired", "Session expired")
	case errors.Is(err, auth.ErrUnauthorized), errors.Is(err, auth.ErrNotFound):
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authorized")
	default:
		log.Printf("auth error: %v", err)
		writeError(w, http.StatusInternalServerError, "unexpected", "Unexpected error")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func withMaxBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.status, time.Since(start).Truncate(time.Millisecond))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
