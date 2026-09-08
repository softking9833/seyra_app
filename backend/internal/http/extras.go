package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"seyra/backend/internal/chat"
	"seyra/backend/internal/media"
)

type editMessageRequest struct {
	Body string `json:"body"`
}

func (s *Server) editMessage(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req editMessageRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	msg, err := s.chat.EditMessage(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("message_id"), req.Body)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messageJSON(msg))
}

func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
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
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	e2e := r.FormValue("e2e") == "true"
	att, err := s.chat.UploadAttachment(r.Context(), actorID, r.PathValue("chat_id"), header.Filename, contentType, bytes.NewReader(payload), int64(len(payload)), e2e)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":              att.ID,
		"conversation_id": att.ConversationID,
		"filename":        att.Filename,
		"content_type":    att.ContentType,
		"byte_size":       att.ByteSize,
		"e2e":             att.E2E,
		"created_at":      att.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"),
	})
}

func (s *Server) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	att, body, size, err := s.chat.OpenAttachment(r.Context(), actorID, r.PathValue("attachment_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", att.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(att.Filename, `"`, "")+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, body)
}

func (s *Server) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	if err := s.chat.DeleteAttachment(r.Context(), actorID, r.PathValue("attachment_id")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	result, err := s.chat.Search(r.Context(), actorID, r.URL.Query().Get("q"), r.URL.Query().Get("type"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := map[string]any{}
	if kind := r.URL.Query().Get("type"); kind == "" || kind == "users" {
		users, err := s.auth.SearchUsers(r.Context(), bearerToken(r), r.URL.Query().Get("q"))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		list := make([]map[string]string, 0, len(users))
		for _, user := range users {
			if !s.profileSearchable(r, actorID, user.ID) {
				continue
			}
			list = append(list, map[string]string{"id": user.ID, "username": user.Username})
		}
		out["users"] = list
	}
	if raw, ok := result["conversations"].([]chat.ConversationSummary); ok {
		chats := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			chats = append(chats, chatSummaryJSON(item))
		}
		out["conversations"] = chats
	}
	if raw, ok := result["messages"].([]chat.Message); ok {
		messages := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			messages = append(messages, messageJSON(item))
		}
		out["messages"] = messages
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) discoverChannels(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	items, err := s.chat.DiscoverChannels(r.Context(), actorID, r.URL.Query().Get("q"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, chatSummaryJSON(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": out})
}

func (s *Server) joinChannel(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	summary, err := s.chat.JoinChannel(r.Context(), actorID, r.PathValue("chat_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, chatSummaryJSON(summary))
}

type reactRequest struct {
	Emoji string `json:"emoji"`
}

func (s *Server) reactToMessage(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req reactRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.chat.React(r.Context(), actorID, r.PathValue("chat_id"), r.PathValue("message_id"), req.Emoji); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listAttachments(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	items, err := s.chat.ListConversationAttachments(r.Context(), actorID, r.PathValue("chat_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, att := range items {
		out = append(out, map[string]any{
			"id":              att.ID,
			"conversation_id": att.ConversationID,
			"filename":        att.Filename,
			"content_type":    att.ContentType,
			"byte_size":       att.ByteSize,
			"e2e":             att.E2E,
			"created_at":      att.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"attachments": out})
}

func (s *Server) listStickers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUser(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"packs": []map[string]any{{
			"id":       "seyra",
			"name":     "Seyra Classic",
			"stickers": chat.BuiltinStickers(),
		}},
	})
}

func (s *Server) peerLastSeen(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	at, err := s.chat.PeerLastSeen(r.Context(), actorID, r.PathValue("user_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := map[string]any{"visible": at != nil}
	if at != nil {
		out["last_seen_at"] = at.UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) chatReceipts(w http.ResponseWriter, r *http.Request) {
	actorID, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	at, err := s.chat.DirectReadReceipt(r.Context(), actorID, r.PathValue("chat_id"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := map[string]any{"visible": at != nil}
	if at != nil {
		out["last_read_at"] = at.UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, out)
}
