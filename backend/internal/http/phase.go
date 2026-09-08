package httpapi

import (
	"net/http"
	"os"
	"strings"
	"time"

	"seyra/backend/internal/auth"
	"seyra/backend/internal/chat"
)

func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	token := bearerToken(r)
	if strings.HasPrefix(token, "bot_") {
		bot, err := s.chat.BotByToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Not authorized")
			return "", false
		}
		return bot.UserID, true
	}
	user, _, err := s.auth.CurrentSession(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return "", false
	}
	_ = s.chat.TouchPresence(r.Context(), user.ID)
	return user.ID, true
}

func (s *Server) listAuthSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.auth.ListSessions(r.Context(), bearerToken(r))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(sessions))
	for _, sess := range sessions {
		item := map[string]any{
			"id":         sess.ID,
			"expires_at": sess.ExpiresAt.UTC().Format(time.RFC3339Nano),
			"created_at": sess.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		if sess.RevokedAt != nil {
			item["revoked"] = true
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": out})
}

func (s *Server) revokeAuthSession(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.RevokeSessionID(r.Context(), bearerToken(r), r.PathValue("session_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getPrivacy(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	settings, err := s.chat.GetPrivacy(r.Context(), actorID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, privacyJSON(settings))
}

func (s *Server) putPrivacy(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		LastSeenVisible     *bool `json:"last_seen_visible"`
		ReadReceipts        *bool `json:"read_receipts"`
		TypingVisible       *bool `json:"typing_visible"`
		ProfileVisible      *bool `json:"profile_visible"`
		NotificationPreview *bool `json:"notification_preview"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	current, err := s.chat.GetPrivacy(r.Context(), actorID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if req.LastSeenVisible != nil {
		current.LastSeenVisible = *req.LastSeenVisible
	}
	if req.ReadReceipts != nil {
		current.ReadReceipts = *req.ReadReceipts
	}
	if req.TypingVisible != nil {
		current.TypingVisible = *req.TypingVisible
	}
	if req.ProfileVisible != nil {
		current.ProfileVisible = *req.ProfileVisible
	}
	if req.NotificationPreview != nil {
		current.NotificationPreview = *req.NotificationPreview
	}
	if err := s.chat.PutPrivacy(r.Context(), actorID, current); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, privacyJSON(current))
}

func privacyJSON(settings chat.PrivacySettings) map[string]any {
	return map[string]any{
		"last_seen_visible":    settings.LastSeenVisible,
		"read_receipts":        settings.ReadReceipts,
		"typing_visible":       settings.TypingVisible,
		"profile_visible":      settings.ProfileVisible,
		"notification_preview": settings.NotificationPreview,
	}
}

func (s *Server) blockUser(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.chat.Block(r.Context(), actorID, req.Username); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unblockUser(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.chat.Unblock(r.Context(), actorID, r.PathValue("user_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listBlocked(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	users, err := s.chat.ListBlocked(r.Context(), actorID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]string, 0, len(users))
	for _, user := range users {
		out = append(out, map[string]string{"id": user.ID, "username": user.Username})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (s *Server) reportUser(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
		Reason   string `json:"reason"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.chat.Report(r.Context(), actorID, req.Username, req.Reason); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pinMessage(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.chat.Pin(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("message_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unpinMessage(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.chat.Unpin(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("message_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listPins(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	messages, err := s.chat.ListPinned(r.Context(), actorID, r.PathValue("chat_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		out = append(out, messageJSON(msg))
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}

func (s *Server) putMute(w http.ResponseWriter, r *http.Request) {
	s.setFlag(w, r, func(actor, chatID string, value bool) error {
		return s.chat.SetMuted(r.Context(), actor, chatID, value)
	})
}

func (s *Server) putArchive(w http.ResponseWriter, r *http.Request) {
	s.setFlag(w, r, func(actor, chatID string, value bool) error {
		return s.chat.SetArchived(r.Context(), actor, chatID, value)
	})
}

func (s *Server) setFlag(w http.ResponseWriter, r *http.Request, fn func(actor, chatID string, value bool) error) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Value bool `json:"value"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := fn(actorID, r.PathValue("chat_id"), req.Value); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) putDraft(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.chat.SaveDraft(r.Context(), actorID, r.PathValue("chat_id"), req.Body); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getDraft(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	body, err := s.chat.GetDraft(r.Context(), actorID, r.PathValue("chat_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"body": body})
}

func (s *Server) createInvite(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	link, err := s.chat.CreateInvite(r.Context(), actorID, r.PathValue("chat_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":    link.ID,
		"token": link.Token,
	})
}

func (s *Server) joinInvite(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	summary, err := s.chat.JoinInvite(r.Context(), actorID, req.Token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, chatSummaryJSON(summary))
}

func (s *Server) patchRoom(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.chat.UpdateRoom(r.Context(), actorID, r.PathValue("chat_id"), req.Title, req.Description, req.Visibility); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) transferOwnership(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.chat.TransferOwnership(r.Context(), actorID, r.PathValue("chat_id"), req.UserID); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restrictMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		CanSend *bool `json:"can_send"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	canSend := true
	if req.CanSend != nil {
		canSend = *req.CanSend
	}
	if err := s.chat.RestrictMember(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("user_id"), canSend); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) forwardMessage(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		DestinationID string `json:"destination_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	msg, err := s.chat.ForwardMessage(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("message_id"), req.DestinationID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, messageJSON(msg))
}

func (s *Server) publishKeys(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		DeviceID           string `json:"device_id"`
		RegistrationID     int    `json:"registration_id"`
		IdentityPublic     string `json:"identity_public"`
		SignedPreKeyID     int    `json:"signed_prekey_id"`
		SignedPreKeyPublic string `json:"signed_prekey_public"`
		SignedPreKeySig    string `json:"signed_prekey_sig"`
		PreKeys            []struct {
			KeyID     int    `json:"key_id"`
			PublicKey string `json:"public_key"`
		} `json:"prekeys"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	dev := chat.E2EDevice{
		DeviceID: req.DeviceID, RegistrationID: req.RegistrationID,
		IdentityPublic: req.IdentityPublic, SignedPreKeyID: req.SignedPreKeyID,
		SignedPreKeyPublic: req.SignedPreKeyPublic, SignedPreKeySig: req.SignedPreKeySig,
	}
	keys := make([]chat.E2EPreKey, 0, len(req.PreKeys))
	for _, item := range req.PreKeys {
		keys = append(keys, chat.E2EPreKey{KeyID: item.KeyID, PublicKey: item.PublicKey})
	}
	if err := s.chat.PublishKeys(r.Context(), actorID, dev, keys); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) fetchBundle(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	dev, pk, err := s.chat.FetchBundle(r.Context(), actorID, r.PathValue("user_id"), r.URL.Query().Get("device_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id":              dev.DeviceID,
		"registration_id":        dev.RegistrationID,
		"identity_public":        dev.IdentityPublic,
		"signed_prekey_id":       dev.SignedPreKeyID,
		"signed_prekey_public":   dev.SignedPreKeyPublic,
		"signed_prekey_sig":      dev.SignedPreKeySig,
		"one_time_prekey_id":     pk.KeyID,
		"one_time_prekey_public": pk.PublicKey,
	})
}

func (s *Server) listE2EDevices(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	devs, err := s.chat.ListE2EDevices(r.Context(), actorID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(devs))
	for _, dev := range devs {
		out = append(out, map[string]any{
			"device_id":       dev.DeviceID,
			"registration_id": dev.RegistrationID,
			"identity_public": dev.IdentityPublic,
			"created_at":      dev.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": out})
}

func (s *Server) revokeE2EDevice(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.chat.RevokeE2EDevice(r.Context(), actorID, r.PathValue("device_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) iceConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	stuns := strings.Split(strings.TrimSpace(os.Getenv("SEYRA_STUN_URLS")), ",")
	urls := make([]string, 0, len(stuns))
	for _, item := range stuns {
		item = strings.TrimSpace(item)
		if item != "" {
			urls = append(urls, item)
		}
	}
	if len(urls) == 0 {
		urls = []string{"stun:stun.l.google.com:19302"}
	}
	ice := []map[string]any{{"urls": urls}}
	turn := strings.TrimSpace(os.Getenv("SEYRA_TURN_URL"))
	if turn != "" {
		ice = append(ice, map[string]any{
			"urls":       []string{turn},
			"username":   os.Getenv("SEYRA_TURN_USERNAME"),
			"credential": os.Getenv("SEYRA_TURN_CREDENTIAL"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ice_servers": ice})
}

func (s *Server) startCallHTTP(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		ConversationID string         `json:"conversation_id"`
		Kind           string         `json:"kind"`
		Payload        map[string]any `json:"payload"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	call, err := s.chat.StartCall(r.Context(), actorID, req.ConversationID, req.Kind, req.Payload)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":              call.ID,
		"conversation_id": call.ConversationID,
		"caller_id":       call.CallerID,
		"kind":            call.Kind,
		"state":           call.State,
		"created_at":      call.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (s *Server) signalCallHTTP(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Action  string         `json:"action"`
		Payload map[string]any `json:"payload"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.chat.SignalCall(r.Context(), actorID, r.PathValue("call_id"), req.Action, req.Payload); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listCalls(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	calls, err := s.chat.ListCalls(r.Context(), actorID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		item := map[string]any{
			"id":              call.ID,
			"conversation_id": call.ConversationID,
			"caller_id":       call.CallerID,
			"kind":            call.Kind,
			"state":           call.State,
			"created_at":      call.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		if call.EndedAt != nil {
			item["ended_at"] = call.EndedAt.UTC().Format(time.RFC3339Nano)
			item["duration_seconds"] = int(call.EndedAt.Sub(call.CreatedAt).Seconds())
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"calls": out})
}

func (s *Server) createBot(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := s.auth.CreateBotUser(r.Context(), req.Username)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	bot, token, err := s.chat.CreateBot(r.Context(), actorID, user.ID, user.Username)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         bot.ID,
		"user_id":    bot.UserID,
		"username":   bot.Username,
		"token":      token,
		"created_at": bot.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (s *Server) listBots(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	bots, err := s.chat.ListBots(r.Context(), actorID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(bots))
	for _, bot := range bots {
		out = append(out, map[string]any{
			"id":       bot.ID,
			"user_id":  bot.UserID,
			"username": bot.Username,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"bots": out})
}

func (s *Server) deleteBot(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	bot, err := s.chat.DeleteBot(r.Context(), actorID, r.PathValue("bot_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	_ = s.auth.DeleteUserRecord(r.Context(), bot.UserID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) grantBot(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		ConversationID    string `json:"conversation_id"`
		CanRead           *bool  `json:"can_read"`
		CanSend           *bool  `json:"can_send"`
		CanManageMessages *bool  `json:"can_manage_messages"`
		CanManageMembers  *bool  `json:"can_manage_members"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	grant := chat.BotGrant{CanRead: true, CanSend: true}
	if req.CanRead != nil {
		grant.CanRead = *req.CanRead
	}
	if req.CanSend != nil {
		grant.CanSend = *req.CanSend
	}
	if req.CanManageMessages != nil {
		grant.CanManageMessages = *req.CanManageMessages
	}
	if req.CanManageMembers != nil {
		grant.CanManageMembers = *req.CanManageMembers
	}
	if err := s.chat.GrantBot(r.Context(), actorID, r.PathValue("bot_id"), req.ConversationID, grant); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) seedDevBot(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if strings.EqualFold(os.Getenv("SEYRA_ENV"), "production") {
		writeError(w, http.StatusForbidden, "forbidden", "Not allowed")
		return
	}
	user, err := s.auth.CreateBotUser(r.Context(), "echo_bot")
	if err != nil && !strings.Contains(err.Error(), "taken") {
		if err != auth.ErrUsernameTaken {
			writeAuthError(w, err)
			return
		}
	}
	if user.ID == "" {
		writeError(w, http.StatusConflict, "username_taken", "Dev bot already exists; use your existing echo_bot")
		return
	}
	bot, token, err := s.chat.CreateBot(r.Context(), actorID, user.ID, user.Username)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       bot.ID,
		"username": bot.Username,
		"token":    token,
		"hint":     "Grant this bot into a group, then send /ping",
	})
}
