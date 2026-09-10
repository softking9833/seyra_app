package httpapi

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"seyra/backend/internal/auth"
	"seyra/backend/internal/chat"
	"seyra/backend/internal/media"
	"seyra/backend/internal/notify"
)

type Server struct {
	auth   *auth.Service
	chat   *chat.Service
	hub    *chat.Hub
	alerts *notify.Service
}

func NewServer(authService *auth.Service, chatService *chat.Service, hub *chat.Hub, alerts *notify.Service) http.Handler {
	s := &Server{auth: authService, chat: chatService, hub: hub, alerts: alerts}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /v1/auth/register", s.register)
	mux.HandleFunc("POST /v1/auth/login", s.login)
	mux.HandleFunc("GET /v1/auth/session", s.session)
	mux.HandleFunc("POST /v1/auth/logout", s.logout)
	mux.HandleFunc("POST /v1/auth/refresh", s.refresh)
	mux.HandleFunc("POST /v1/auth/account/delete", s.deleteAccount)
	mux.HandleFunc("GET /v1/auth/sessions", s.listAuthSessions)
	mux.HandleFunc("DELETE /v1/auth/sessions/{session_id}", s.revokeAuthSession)
	mux.HandleFunc("GET /v1/users/me", s.me)
	mux.HandleFunc("PATCH /v1/users/me", s.patchMe)
	mux.HandleFunc("PUT /v1/users/me", s.patchMe)
	mux.HandleFunc("PATCH /v1/users/me/username", s.patchUsername)
	mux.HandleFunc("PUT /v1/users/me/username", s.patchUsername)
	mux.HandleFunc("POST /v1/users/me/avatar", s.uploadAvatar)
	mux.HandleFunc("DELETE /v1/users/me/avatar", s.deleteAvatar)
	mux.HandleFunc("GET /v1/users/{user_id}/avatar", s.getAvatar)
	mux.HandleFunc("POST /v1/auth/sessions/others", s.revokeOtherSessions)
	mux.HandleFunc("GET /v1/users/search", s.searchUsers)
	mux.HandleFunc("POST /v1/chats", s.createChat)
	mux.HandleFunc("POST /v1/chats/groups", s.createGroup)
	mux.HandleFunc("POST /v1/chats/channels", s.createChannel)
	mux.HandleFunc("GET /v1/chats", s.listChats)
	mux.HandleFunc("GET /v1/chats/{chat_id}/messages", s.listMessages)
	mux.HandleFunc("POST /v1/chats/{chat_id}/messages", s.sendMessage)
	mux.HandleFunc("DELETE /v1/chats/{chat_id}/messages/{message_id}", s.deleteMessage)
	mux.HandleFunc("GET /v1/chats/{chat_id}/members", s.listMembers)
	mux.HandleFunc("POST /v1/chats/{chat_id}/members", s.addMembers)
	mux.HandleFunc("DELETE /v1/chats/{chat_id}/members/{user_id}", s.removeMember)
	mux.HandleFunc("POST /v1/chats/{chat_id}/members/{user_id}/role", s.setMemberRole)
	mux.HandleFunc("POST /v1/chats/{chat_id}/leave", s.leaveChat)
	mux.HandleFunc("PATCH /v1/chats/{chat_id}", s.patchRoom)
	mux.HandleFunc("POST /v1/chats/{chat_id}/transfer", s.transferOwnership)
	mux.HandleFunc("POST /v1/chats/{chat_id}/members/{user_id}/restrict", s.restrictMember)
	mux.HandleFunc("PUT /v1/chats/{chat_id}/mute", s.putMute)
	mux.HandleFunc("PUT /v1/chats/{chat_id}/archive", s.putArchive)
	mux.HandleFunc("GET /v1/chats/{chat_id}/draft", s.getDraft)
	mux.HandleFunc("PUT /v1/chats/{chat_id}/draft", s.putDraft)
	mux.HandleFunc("POST /v1/chats/{chat_id}/invites", s.createInvite)
	mux.HandleFunc("POST /v1/invites/join", s.joinInvite)
	mux.HandleFunc("GET /v1/chats/{chat_id}/pins", s.listPins)
	mux.HandleFunc("POST /v1/chats/{chat_id}/messages/{message_id}/pin", s.pinMessage)
	mux.HandleFunc("DELETE /v1/chats/{chat_id}/messages/{message_id}/pin", s.unpinMessage)
	mux.HandleFunc("POST /v1/chats/{chat_id}/messages/{message_id}/forward", s.forwardMessage)
	mux.HandleFunc("PATCH /v1/chats/{chat_id}/messages/{message_id}", s.editMessage)
	mux.HandleFunc("POST /v1/chats/{chat_id}/join", s.joinChannel)
	mux.HandleFunc("POST /v1/chats/{chat_id}/attachments", s.uploadAttachment)
	mux.HandleFunc("GET /v1/chats/{chat_id}/attachments", s.listAttachments)
	mux.HandleFunc("GET /v1/attachments/{attachment_id}", s.downloadAttachment)
	mux.HandleFunc("DELETE /v1/attachments/{attachment_id}", s.deleteAttachment)
	mux.HandleFunc("GET /v1/stickers", s.listStickers)
	mux.HandleFunc("GET /v1/users/{user_id}/last-seen", s.peerLastSeen)
	mux.HandleFunc("GET /v1/chats/{chat_id}/receipts", s.chatReceipts)
	mux.HandleFunc("POST /v1/chats/{chat_id}/read", s.markChatRead)
	mux.HandleFunc("PUT /v1/chats/{chat_id}/messages/{message_id}/reactions", s.reactToMessage)
	mux.HandleFunc("GET /v1/search", s.search)
	mux.HandleFunc("GET /v1/channels/discover", s.discoverChannels)
	mux.HandleFunc("GET /v1/privacy", s.getPrivacy)
	mux.HandleFunc("PUT /v1/privacy", s.putPrivacy)
	mux.HandleFunc("GET /v1/blocks", s.listBlocked)
	mux.HandleFunc("POST /v1/blocks", s.blockUser)
	mux.HandleFunc("DELETE /v1/blocks/{user_id}", s.unblockUser)
	mux.HandleFunc("POST /v1/reports", s.reportUser)
	mux.HandleFunc("POST /v1/e2e/keys", s.publishKeys)
	mux.HandleFunc("GET /v1/e2e/devices", s.listE2EDevices)
	mux.HandleFunc("DELETE /v1/e2e/devices/{device_id}", s.revokeE2EDevice)
	mux.HandleFunc("GET /v1/e2e/bundle/{user_id}", s.fetchBundle)
	mux.HandleFunc("GET /v1/calls/ice", s.iceConfig)
	mux.HandleFunc("POST /v1/calls", s.startCallHTTP)
	mux.HandleFunc("POST /v1/calls/{call_id}/signal", s.signalCallHTTP)
	mux.HandleFunc("GET /v1/calls", s.listCalls)
	mux.HandleFunc("DELETE /v1/calls", s.clearCalls)
	mux.HandleFunc("DELETE /v1/calls/{call_id}", s.deleteCall)
	mux.HandleFunc("POST /v1/bots", s.createBot)
	mux.HandleFunc("GET /v1/bots", s.listBots)
	mux.HandleFunc("DELETE /v1/bots/{bot_id}", s.deleteBot)
	mux.HandleFunc("POST /v1/bots/{bot_id}/grants", s.grantBot)
	mux.HandleFunc("POST /v1/bots/dev/echo", s.seedDevBot)
	mux.HandleFunc("POST /v1/notifications/devices", s.registerDevice)
	mux.HandleFunc("DELETE /v1/notifications/devices/{device_id}", s.unregisterDevice)
	mux.HandleFunc("GET /v1/notifications/preferences", s.getNotificationPreferences)
	mux.HandleFunc("PUT /v1/notifications/preferences", s.putNotificationPreferences)
	mux.HandleFunc("GET /v1/realtime", s.realtime)
	return withLogging(withCORS(withMaxBody(mux)))
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

type deleteAccountRequest struct {
	Password string `json:"password"`
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
	s.auth.AnnotateSession(r.Context(), issued.AccessToken, r.UserAgent())
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
	s.auth.AnnotateSession(r.Context(), issued.AccessToken, r.UserAgent())
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

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	user, _, err := s.auth.CurrentSession(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicProfile(user))
}

func (s *Server) searchUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.auth.SearchUsers(r.Context(), bearerToken(r), r.URL.Query().Get("q"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	actorID := ""
	if user, _, err := s.auth.CurrentSession(r.Context(), bearerToken(r)); err == nil {
		actorID = user.ID
	}
	out := make([]map[string]string, 0, len(users))
	for _, user := range users {
		if !s.profileSearchable(r, actorID, user.ID) {
			continue
		}
		out = append(out, map[string]string{
			"id":       user.ID,
			"username": user.Username,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (s *Server) profileSearchable(r *http.Request, actorID, userID string) bool {
	if userID == actorID {
		return true
	}
	priv, err := s.chat.GetPrivacy(r.Context(), userID)
	if err != nil {
		return true
	}
	return priv.ProfileVisible
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var req deleteAccountRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, _, err := s.auth.CurrentSession(r.Context(), bearerToken(r))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeAuthError(w, auth.ErrInvalidCredentials)
		return
	}
	ok, err := auth.ComparePassword(user.PasswordHash, req.Password)
	if err != nil || !ok {
		writeAuthError(w, auth.ErrInvalidCredentials)
		return
	}
	bots, err := s.chat.ListBots(r.Context(), user.ID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	for _, bot := range bots {
		if _, err := s.chat.DeleteBot(r.Context(), user.ID, bot.ID); err != nil {
			writeAuthError(w, err)
			return
		}
		if err := s.auth.DeleteUserRecord(r.Context(), bot.UserID); err != nil {
			writeAuthError(w, err)
			return
		}
	}
	if err := s.auth.DeleteAccount(r.Context(), bearerToken(r), req.Password); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	if err := decoder.Decode(dest); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid request")
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

func publicProfile(user auth.User) map[string]any {
	return map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"bio":          user.Bio,
		"has_avatar":   user.AvatarKey != "",
		"created_at":   user.CreatedAt.UTC().Format(time.RFC3339Nano),
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
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid request")
	case errors.Is(err, auth.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "username_taken", "Username is already taken")
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
	case errors.Is(err, auth.ErrSessionExpired):
		writeError(w, http.StatusUnauthorized, "session_expired", "Session expired")
	case errors.Is(err, auth.ErrRateLimited):
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many attempts")
	case errors.Is(err, auth.ErrUnauthorized), errors.Is(err, auth.ErrNotFound):
		writeError(w, http.StatusUnauthorized, "unauthorized", "Not authorized")
	case errors.Is(err, chat.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid request")
	case errors.Is(err, chat.ErrCannotMessageSelf):
		writeError(w, http.StatusBadRequest, "cannot_message_self", "Cannot start a conversation with yourself")
	case errors.Is(err, chat.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, chat.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Not allowed")
	case errors.Is(err, chat.ErrAlreadyMember):
		writeError(w, http.StatusConflict, "already_member", "User is already a member")
	case errors.Is(err, chat.ErrOwnerProtected):
		writeError(w, http.StatusForbidden, "owner_protected", "The owner cannot be removed or demoted")
	case errors.Is(err, media.ErrInvalidType):
		writeError(w, http.StatusBadRequest, "invalid_type", "File type is not allowed")
	case errors.Is(err, media.ErrTooLarge):
		writeError(w, http.StatusBadRequest, "too_large", "File is too large")
	case errors.Is(err, media.ErrInvalidKey), errors.Is(err, media.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Not found")
	default:
		log.Printf("internal auth error")
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

func withCORS(next http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	raw := strings.TrimSpace(os.Getenv("SEYRA_CORS_ORIGINS"))
	if raw == "" {
		raw = "http://172.20.1.78:5000"
	}
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withMaxBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := int64(16 * 1024)
		if r.Method == http.MethodPost {
			path := r.URL.Path
			if strings.HasSuffix(path, "/attachments") || strings.HasSuffix(path, "/avatar") {
				limit = 26 << 20
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
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

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return hijacker.Hijack()
}
