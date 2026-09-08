package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"seyra/backend/internal/chat"
)

type createChatRequest struct {
	Username string `json:"username"`
}

type createRoomRequest struct {
	Title      string   `json:"title"`
	Usernames  []string `json:"usernames"`
	Visibility string   `json:"visibility"`
}

type sendMessageRequest struct {
	Body         string `json:"body"`
	ReplyToID    string `json:"reply_to_id"`
	AttachmentID string `json:"attachment_id"`
	E2E          bool   `json:"e2e"`
}

func (s *Server) createChat(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req createChatRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	summary, err := s.chat.CreateDirect(r.Context(), actorID, req.Username)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, chatSummaryJSON(summary))
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	s.createRoom(w, r, chat.KindGroup)
}

func (s *Server) createChannel(w http.ResponseWriter, r *http.Request) {
	s.createRoom(w, r, chat.KindChannel)
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request, kind string) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req createRoomRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	summary, err := s.chat.CreateRoom(r.Context(), actorID, kind, req.Title, req.Usernames, req.Visibility)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, chatSummaryJSON(summary))
}

func (s *Server) listChats(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	items, err := s.chat.List(r.Context(), actorID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, chatSummaryJSON(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"chats": out})
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	chatID := r.PathValue("chat_id")
	before := r.URL.Query().Get("before")
	limit := chat.DefaultPageSize
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "Invalid request")
			return
		}
		limit = parsed
	}
	messages, err := s.chat.ListMessages(r.Context(), actorID, chatID, before, limit)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		out = append(out, messageJSON(msg))
	}
	var nextBefore string
	if len(messages) > 0 {
		nextBefore = messages[0].ID
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"messages":    out,
		"next_before": nextBefore,
	})
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req sendMessageRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	msg, err := s.chat.SendMessage(r.Context(), actorID, r.PathValue("chat_id"), req.Body, req.ReplyToID, req.AttachmentID, req.E2E)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	s.dispatchMessageAlert(r, actorID, msg)
	writeJSON(w, http.StatusCreated, messageJSON(msg))
}

func (s *Server) dispatchMessageAlert(r *http.Request, actorID string, msg chat.Message) {
	if s.alerts == nil {
		return
	}
	members, err := s.chat.MemberIDs(r.Context(), msg.ConversationID)
	if err != nil {
		return
	}
	name := "Seyra"
	if user, _, err := s.auth.CurrentSession(r.Context(), bearerToken(r)); err == nil {
		name = user.Username
	}
	for _, memberID := range members {
		if memberID == actorID {
			continue
		}
		preview := msg.Body
		if msg.E2E {
			preview = "Encrypted message"
		} else {
			priv, err := s.chat.GetPrivacy(r.Context(), memberID)
			if err == nil && !priv.NotificationPreview {
				preview = "New message"
			}
		}
		s.alerts.NotifyNewMessage(r.Context(), actorID, name, msg.ConversationID, msg.ID, preview, []string{memberID})
	}
}

func (s *Server) deleteMessage(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.chat.DeleteMessage(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("message_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type addMembersRequest struct {
	Usernames []string `json:"usernames"`
}

type setRoleRequest struct {
	Role string `json:"role"`
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	members, err := s.chat.ListRoomMembers(r.Context(), actorID, r.PathValue("chat_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(members))
	for _, m := range members {
		out = append(out, map[string]any{
			"id":       m.User.ID,
			"username": m.User.Username,
			"role":     m.Role,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": out})
}

func (s *Server) addMembers(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req addMembersRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	members, err := s.chat.AddMembers(r.Context(), actorID, r.PathValue("chat_id"), req.Usernames)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(members))
	for _, m := range members {
		out = append(out, map[string]any{
			"id":       m.User.ID,
			"username": m.User.Username,
			"role":     m.Role,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": out})
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.chat.RemoveMember(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("user_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) leaveChat(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.chat.LeaveRoom(r.Context(), actorID, r.PathValue("chat_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setMemberRole(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req setRoleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.chat.SetMemberRole(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("user_id"), req.Role); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func chatSummaryJSON(item chat.ConversationSummary) map[string]any {
	kind := item.Kind
	if kind == "" {
		kind = chat.KindDirect
	}
	title := item.Title
	if title == "" && kind == chat.KindDirect {
		title = item.Peer.Username
	}
	visibility := item.Visibility
	if visibility == "" {
		visibility = chat.VisibilityPrivate
	}
	return map[string]any{
		"id":           item.ID,
		"kind":         kind,
		"title":        title,
		"member_count": item.MemberCount,
		"visibility":   visibility,
		"peer": map[string]string{
			"id":       item.Peer.ID,
			"username": item.Peer.Username,
		},
		"last_message_preview": item.LastMessagePreview,
		"last_message_at":      item.LastMessageAt.UTC().Format(time.RFC3339Nano),
		"unread_count":         item.UnreadCount,
		"created_at":           item.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func messageJSON(msg chat.Message) map[string]any {
	out := map[string]any{
		"id":              msg.ID,
		"conversation_id": msg.ConversationID,
		"sender_id":       msg.SenderID,
		"body":            msg.Body,
		"created_at":      msg.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if msg.ReplyToID != "" {
		out["reply_to_id"] = msg.ReplyToID
	}
	if msg.AttachmentID != "" {
		out["attachment_id"] = msg.AttachmentID
	}
	if msg.E2E {
		out["e2e"] = true
	}
	if msg.ForwardedFromID != "" {
		out["forwarded_from_id"] = msg.ForwardedFromID
	}
	if msg.EditedAt != nil {
		out["edited_at"] = msg.EditedAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}
