package chat

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"seyra/backend/internal/media"
)

func (s *Service) EditMessage(ctx context.Context, actorID, conversationID, messageID, body string) (Message, error) {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return Message{}, err
	}
	text := strings.TrimSpace(body)
	if text == "" || utf8.RuneCountInString(text) > MaxMessageRunes {
		return Message{}, ErrInvalidInput
	}
	msg, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return Message{}, err
	}
	if msg.ConversationID != conversationID || msg.SenderID != actorID {
		return Message{}, ErrForbidden
	}
	if msg.E2E {
		return Message{}, ErrForbidden
	}
	now := s.now().UTC()
	if err := s.store.UpdateMessageBody(ctx, messageID, text, now); err != nil {
		return Message{}, err
	}
	msg.Body = text
	msg.EditedAt = &now
	s.publishMembers(ctx, conversationID, Event{
		Type: EventMessageUpdated,
		Payload: map[string]any{
			"id":              msg.ID,
			"conversation_id": conversationID,
			"body":            text,
			"edited_at":       now.Format(time.RFC3339Nano),
		},
	})
	return msg, nil
}

func (s *Service) UploadAttachment(ctx context.Context, actorID, conversationID, filename, contentType string, r io.Reader, size int64, e2e bool) (Attachment, error) {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return Attachment{}, err
	}
	if can, err := s.store.MemberCanSend(ctx, conversationID, actorID); err != nil {
		return Attachment{}, err
	} else if !can {
		return Attachment{}, ErrForbidden
	}
	if err := s.requireBotCanSend(ctx, conversationID, actorID); err != nil {
		return Attachment{}, err
	}
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return Attachment{}, err
	}
	if e2e && conv.Kind != KindDirect {
		return Attachment{}, ErrForbidden
	}
	if e2e {
		contentType = "application/octet-stream"
		filename = "encrypted.bin"
	}
	if !media.AllowedContentType(contentType) {
		return Attachment{}, media.ErrInvalidType
	}
	if size <= 0 || size > media.MaxBytes {
		return Attachment{}, media.ErrTooLarge
	}
	safeName := filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if safeName == "" || safeName == "." || safeName == ".." {
		return Attachment{}, ErrInvalidInput
	}
	id := newID("att")
	key := media.KeyFor(id, safeName)
	if err := s.blobs.Put(ctx, key, contentType, r, size); err != nil {
		return Attachment{}, err
	}
	att := Attachment{
		ID:             id,
		ConversationID: conversationID,
		UploaderID:     actorID,
		ObjectKey:      key,
		Filename:       safeName,
		ContentType:    contentType,
		ByteSize:       size,
		E2E:            e2e,
		CreatedAt:      s.now().UTC(),
	}
	if err := s.store.InsertAttachment(ctx, att); err != nil {
		_ = s.blobs.Delete(ctx, key)
		return Attachment{}, err
	}
	return att, nil
}

func (s *Service) OpenAttachment(ctx context.Context, actorID, attachmentID string) (Attachment, io.ReadCloser, int64, error) {
	att, err := s.store.GetAttachment(ctx, attachmentID)
	if err != nil {
		return Attachment{}, nil, 0, err
	}
	if err := s.requireReadable(ctx, att.ConversationID, actorID); err != nil {
		return Attachment{}, nil, 0, err
	}
	body, size, err := s.blobs.Open(ctx, att.ObjectKey)
	if err != nil {
		return Attachment{}, nil, 0, err
	}
	return att, body, size, nil
}

func AvatarKey(userID string) string {
	safe := strings.ReplaceAll(userID, "_", "")
	return media.KeyFor("ava"+safe, "jpg")
}

func (s *Service) PutAvatarBytes(ctx context.Context, userID, contentType string, r io.Reader, size int64) (string, error) {
	if !strings.HasPrefix(contentType, "image/") || !media.AllowedContentType(contentType) {
		return "", media.ErrInvalidType
	}
	if size <= 0 || size > media.MaxBytes {
		return "", media.ErrTooLarge
	}
	key := AvatarKey(userID)
	if err := s.blobs.Put(ctx, key, contentType, r, size); err != nil {
		return "", err
	}
	return key, nil
}

func (s *Service) OpenAvatarBytes(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	return s.blobs.Open(ctx, key)
}

func (s *Service) DeleteAvatarBytes(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return s.blobs.Delete(ctx, key)
}

func (s *Service) DeleteAttachment(ctx context.Context, actorID, attachmentID string) error {
	att, err := s.store.GetAttachment(ctx, attachmentID)
	if err != nil {
		return err
	}
	if att.UploaderID != actorID {
		if err := s.requireAdmin(ctx, att.ConversationID, actorID); err != nil {
			return err
		}
	} else if err := s.requireMember(ctx, att.ConversationID, actorID); err != nil {
		return err
	}
	if err := s.store.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}
	return s.blobs.Delete(ctx, att.ObjectKey)
}

func (s *Service) Search(ctx context.Context, actorID, query, kind string) (map[string]any, error) {
	q := strings.TrimSpace(query)
	if utf8.RuneCountInString(q) < 2 {
		return nil, ErrInvalidInput
	}
	limit := SearchLimit
	out := map[string]any{}
	if kind == "" || kind == "chats" || kind == "groups" || kind == "channels" {
		items, err := s.store.SearchConversations(ctx, actorID, q, limit)
		if err != nil {
			return nil, err
		}
		filtered := make([]ConversationSummary, 0, len(items))
		for _, item := range items {
			if kind == "groups" && item.Kind != KindGroup {
				continue
			}
			if kind == "channels" && item.Kind != KindChannel {
				continue
			}
			if kind == "chats" && item.Kind != KindDirect {
				continue
			}
			filtered = append(filtered, item)
		}
		out["conversations"] = filtered
	}
	if kind == "" || kind == "messages" {
		messages, err := s.store.SearchMessages(ctx, actorID, q, limit)
		if err != nil {
			return nil, err
		}
		readable := make([]Message, 0, len(messages))
		for _, item := range messages {
			if err := s.requireReadable(ctx, item.ConversationID, actorID); err != nil {
				continue
			}
			readable = append(readable, item)
		}
		out["messages"] = readable
	}
	if kind == "" || kind == "users" {
		// Username hits are filled by HTTP using the auth directory so private
		// profile flags can be applied without duplicating user search here.
	}
	return out, nil
}

func (s *Service) DiscoverChannels(ctx context.Context, actorID, query string) ([]ConversationSummary, error) {
	if actorID == "" {
		return nil, ErrForbidden
	}
	return s.store.DiscoverChannels(ctx, query, SearchLimit)
}

func (s *Service) JoinChannel(ctx context.Context, actorID, conversationID string) (ConversationSummary, error) {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return ConversationSummary{}, err
	}
	if conv.Kind != KindChannel || conv.Visibility != VisibilityPublic {
		return ConversationSummary{}, ErrForbidden
	}
	if err := s.store.AddMember(ctx, conversationID, actorID, RoleMember, s.now().UTC()); err != nil && !errors.Is(err, ErrAlreadyMember) {
		return ConversationSummary{}, err
	}
	return s.summaryFor(ctx, actorID, conversationID, UserRef{})
}

func (s *Service) ListConversationAttachments(ctx context.Context, actorID, conversationID string) ([]Attachment, error) {
	if err := s.requireReadable(ctx, conversationID, actorID); err != nil {
		return nil, err
	}
	return s.store.ListAttachments(ctx, conversationID, 100)
}

func (s *Service) TouchPresence(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrForbidden
	}
	return s.store.TouchPresence(ctx, userID, s.now().UTC())
}

func (s *Service) PeerLastSeen(ctx context.Context, actorID, userID string) (*time.Time, error) {
	if actorID == "" || userID == "" {
		return nil, ErrForbidden
	}
	if actorID != userID {
		priv, err := s.store.GetPrivacy(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !priv.LastSeenVisible {
			return nil, nil
		}
		if _, err := s.store.GetConversationByPairKey(ctx, pairKey(actorID, userID)); err != nil {
			return nil, ErrForbidden
		}
	}
	at, err := s.store.GetPresence(ctx, userID)
	if err != nil {
		return nil, nil
	}
	return &at, nil
}

func (s *Service) DirectReadReceipt(ctx context.Context, actorID, conversationID string) (*time.Time, error) {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return nil, err
	}
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Kind != KindDirect {
		return nil, ErrInvalidInput
	}
	members, err := s.store.ListMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	var peer Member
	for _, m := range members {
		if m.UserID != actorID {
			peer = m
			break
		}
	}
	if peer.UserID == "" {
		return nil, ErrNotFound
	}
	priv, err := s.store.GetPrivacy(ctx, peer.UserID)
	if err != nil {
		return nil, err
	}
	if !priv.ReadReceipts {
		return nil, nil
	}
	return peer.LastReadAt, nil
}

func (s *Service) PublishTyping(ctx context.Context, actorID, conversationID string, typing bool) error {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return err
	}
	priv, err := s.store.GetPrivacy(ctx, actorID)
	if err != nil {
		return err
	}
	if !priv.TypingVisible {
		return nil
	}
	s.publishMembers(ctx, conversationID, Event{
		Type: EventTyping,
		Payload: map[string]any{
			"conversation_id": conversationID,
			"user_id":         actorID,
			"typing":          typing,
		},
	})
	return nil
}

type Sticker struct {
	Pack  string `json:"pack"`
	ID    string `json:"id"`
	Emoji string `json:"emoji"`
	Name  string `json:"name"`
}

func BuiltinStickers() []Sticker {
	return []Sticker{
		{Pack: "seyra", ID: "wave", Emoji: "👋", Name: "Wave"},
		{Pack: "seyra", ID: "heart", Emoji: "💙", Name: "Heart"},
		{Pack: "seyra", ID: "fire", Emoji: "🔥", Name: "Fire"},
		{Pack: "seyra", ID: "thumb", Emoji: "👍", Name: "Thumbs up"},
		{Pack: "seyra", ID: "party", Emoji: "🎉", Name: "Party"},
		{Pack: "seyra", ID: "laugh", Emoji: "😂", Name: "Laugh"},
		{Pack: "seyra", ID: "pray", Emoji: "🙏", Name: "Thanks"},
		{Pack: "seyra", ID: "star", Emoji: "⭐", Name: "Star"},
	}
}

func StickerEmoji(id string) (string, bool) {
	for _, item := range BuiltinStickers() {
		if item.ID == id {
			return item.Emoji, true
		}
	}
	return "", false
}
