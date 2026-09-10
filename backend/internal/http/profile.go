package httpapi

import (
	"bytes"
	"io"
	"net/http"

	"seyra/backend/internal/media"
)

type patchMeRequest struct {
	DisplayName *string `json:"display_name"`
	Bio         *string `json:"bio"`
}

type patchUsernameRequest struct {
	Username string `json:"username"`
}

func (s *Server) patchMe(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	user, _, err := s.auth.CurrentSession(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	var req patchMeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	name := user.DisplayName
	if req.DisplayName != nil {
		name = *req.DisplayName
	}
	bio := user.Bio
	if req.Bio != nil {
		bio = *req.Bio
	}
	updated, err := s.auth.UpdateProfile(r.Context(), token, name, bio)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicProfile(updated))
}

func (s *Server) patchUsername(w http.ResponseWriter, r *http.Request) {
	var req patchUsernameRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	updated, err := s.auth.ChangeUsername(r.Context(), bearerToken(r), req.Username)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicProfile(updated))
}

func (s *Server) revokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.RevokeOtherSessions(r.Context(), bearerToken(r)); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	user, _, err := s.auth.CurrentSession(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if err := r.ParseMultipartForm(media.MaxBytes + 1024); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "File is required")
		return
	}
	defer file.Close()
	payload, err := io.ReadAll(io.LimitReader(file, media.MaxBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid upload")
		return
	}
	contentType := media.SniffImageContentType(payload, header.Header.Get("Content-Type"))
	key, err := s.chat.PutAvatarBytes(r.Context(), user.ID, contentType, bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	updated, err := s.auth.SetAvatarKey(r.Context(), token, key)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicProfile(updated))
}

func (s *Server) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	user, _, err := s.auth.CurrentSession(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	_ = s.chat.DeleteAvatarBytes(r.Context(), user.AvatarKey)
	updated, err := s.auth.SetAvatarKey(r.Context(), token, "")
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicProfile(updated))
}

func (s *Server) getAvatar(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	targetID := r.PathValue("user_id")
	user, err := s.auth.UserByID(r.Context(), targetID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}
	if actorID != targetID {
		priv, err := s.chat.GetPrivacy(r.Context(), targetID)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		if !priv.PhotoVisible {
			writeError(w, http.StatusForbidden, "forbidden", "Not allowed")
			return
		}
	}
	if user.AvatarKey == "" {
		writeError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}
	body, size, err := s.chat.OpenAvatarBytes(r.Context(), user.AvatarKey)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=60")
	w.WriteHeader(http.StatusOK)
	if size > 0 {
		_, _ = io.CopyN(w, body, size)
		return
	}
	_, _ = io.Copy(w, body)
}
