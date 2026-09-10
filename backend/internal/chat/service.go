package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"seyra/backend/internal/auth"
	"seyra/backend/internal/media"
)

type Service struct {
	store     Store
	directory Directory
	realtime  Publisher
	blobs     media.Store
	now       func() time.Time
}

func NewService(store Store, directory Directory, realtime Publisher) *Service {
	return &Service{
		store:     store,
		directory: directory,
		realtime:  realtime,
		blobs:     media.NewMemoryStore(),
		now:       time.Now,
	}
}

func (s *Service) SetBlobStore(store media.Store) {
	if store != nil {
		s.blobs = store
	}
}

func (s *Service) CreateDirect(ctx context.Context, actorID, username string) (ConversationSummary, error) {
	normalized, err := auth.NormalizeUsername(username)
	if err != nil {
		return ConversationSummary{}, ErrInvalidInput
	}
	peer, err := s.directory.LookupUsername(ctx, normalized)
	if err != nil {
		return ConversationSummary{}, err
	}
	if peer.ID == actorID {
		return ConversationSummary{}, ErrCannotMessageSelf
	}
	blocked, err := s.store.IsBlocked(ctx, actorID, peer.ID)
	if err != nil {
		return ConversationSummary{}, err
	}
	if blocked {
		return ConversationSummary{}, ErrForbidden
	}
	key := pairKey(actorID, peer.ID)
	existing, err := s.store.GetConversationByPairKey(ctx, key)
	if err == nil {
		return s.summaryFor(ctx, actorID, existing.ID, peer)
	}
	if !errors.Is(err, ErrNotFound) {
		return ConversationSummary{}, err
	}
	now := s.now().UTC()
	conv := Conversation{
		ID:        newID("cht"),
		PairKey:   key,
		Kind:      KindDirect,
		CreatedAt: now,
	}
	if err := s.store.CreateConversation(ctx, conv, [2]string{actorID, peer.ID}); err != nil {
		return ConversationSummary{}, err
	}
	created, err := s.store.GetConversationByPairKey(ctx, key)
	if err != nil {
		return ConversationSummary{}, err
	}
	return s.summaryFor(ctx, actorID, created.ID, peer)
}

func (s *Service) CreateRoom(ctx context.Context, actorID, kind, title string, usernames []string, visibility string) (ConversationSummary, error) {
	if kind != KindGroup && kind != KindChannel {
		return ConversationSummary{}, ErrInvalidInput
	}
	name := strings.TrimSpace(title)
	if name == "" || utf8.RuneCountInString(name) > MaxTitleRunes {
		return ConversationSummary{}, ErrInvalidInput
	}
	seen := map[string]struct{}{actorID: {}}
	memberIDs := []string{actorID}
	for _, raw := range usernames {
		normalized, err := auth.NormalizeUsername(raw)
		if err != nil {
			return ConversationSummary{}, ErrInvalidInput
		}
		peer, err := s.directory.LookupUsername(ctx, normalized)
		if err != nil {
			return ConversationSummary{}, err
		}
		if _, ok := seen[peer.ID]; ok {
			continue
		}
		seen[peer.ID] = struct{}{}
		memberIDs = append(memberIDs, peer.ID)
	}
	if kind == KindGroup && len(memberIDs) < 2 {
		return ConversationSummary{}, ErrInvalidInput
	}
	if len(memberIDs) > MaxRoomMembers {
		return ConversationSummary{}, ErrInvalidInput
	}
	now := s.now().UTC()
	vis := strings.TrimSpace(visibility)
	if vis == "" {
		vis = VisibilityPrivate
	}
	if kind == KindGroup {
		vis = VisibilityPrivate
	}
	if vis != VisibilityPrivate && vis != VisibilityPublic {
		return ConversationSummary{}, ErrInvalidInput
	}
	conv := Conversation{
		ID:         newID("cht"),
		Kind:       kind,
		Title:      name,
		Visibility: vis,
		CreatedAt:  now,
	}
	if err := s.store.CreateRoom(ctx, conv, actorID, memberIDs); err != nil {
		return ConversationSummary{}, err
	}
	return s.summaryFor(ctx, actorID, conv.ID, UserRef{})
}

func (s *Service) List(ctx context.Context, actorID string) ([]ConversationSummary, error) {
	items, err := s.store.ListSummaries(ctx, actorID)
	if err != nil {
		return nil, err
	}
	for i, item := range items {
		if item.Kind != "" && item.Kind != KindDirect {
			continue
		}
		if item.Peer.Username != "" || item.Peer.ID == "" {
			continue
		}
		peer, err := s.directory.LookupID(ctx, item.Peer.ID)
		if err != nil {
			continue
		}
		items[i].Peer = peer
	}
	return items, nil
}

func (s *Service) ListMessages(ctx context.Context, actorID, conversationID, beforeID string, limit int) ([]Message, error) {
	if err := s.requireReadable(ctx, conversationID, actorID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	messages, err := s.store.ListMessages(ctx, conversationID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	s.markReadAndNotify(ctx, conversationID, actorID)
	return messages, nil
}

func (s *Service) MarkConversationRead(ctx context.Context, actorID, conversationID string) error {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return err
	}
	s.markReadAndNotify(ctx, conversationID, actorID)
	return nil
}

func (s *Service) markReadAndNotify(ctx context.Context, conversationID, actorID string) {
	now := s.now().UTC()
	_ = s.store.MarkRead(ctx, conversationID, actorID, now)
	priv, err := s.store.GetPrivacy(ctx, actorID)
	if err != nil || !priv.ReadReceipts {
		return
	}
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil || conv.Kind != KindDirect {
		return
	}
	s.publishMembers(ctx, conversationID, Event{
		Type: EventReceiptUpdated,
		Payload: map[string]any{
			"conversation_id": conversationID,
			"user_id":         actorID,
			"last_read_at":    now.Format(time.RFC3339Nano),
		},
	})
}

func (s *Service) SendMessage(ctx context.Context, actorID, conversationID, body, replyToID, attachmentID string, e2e bool) (Message, error) {
	return s.sendMessage(ctx, actorID, conversationID, body, replyToID, attachmentID, e2e, "")
}

func (s *Service) sendMessage(ctx context.Context, actorID, conversationID, body, replyToID, attachmentID string, e2e bool, forwardedFrom string) (Message, error) {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return Message{}, err
	}
	if can, err := s.store.MemberCanSend(ctx, conversationID, actorID); err != nil {
		return Message{}, err
	} else if !can {
		return Message{}, ErrForbidden
	}
	text := strings.TrimSpace(body)
	if utf8.RuneCountInString(text) > MaxMessageRunes {
		return Message{}, ErrInvalidInput
	}
	if text == "" && strings.TrimSpace(attachmentID) == "" {
		return Message{}, ErrInvalidInput
	}
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return Message{}, err
	}
	if conv.Kind == KindChannel && e2e {
		return Message{}, ErrForbidden
	}
	if conv.Kind == KindDirect {
		ids, err := s.store.MemberIDs(ctx, conversationID)
		if err != nil {
			return Message{}, err
		}
		for _, id := range ids {
			if id == actorID {
				continue
			}
			blocked, err := s.store.IsBlocked(ctx, actorID, id)
			if err != nil {
				return Message{}, err
			}
			if blocked {
				return Message{}, ErrForbidden
			}
		}
	}
	bot, botErr := s.store.GetBotByUserID(ctx, actorID)
	isBot := botErr == nil
	if isBot {
		grant, err := s.store.GetBotGrant(ctx, bot.ID, conversationID)
		if err != nil || !grant.CanSend {
			return Message{}, ErrForbidden
		}
	}
	if conv.Kind == KindChannel && !isBot {
		if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
			return Message{}, err
		}
	}
	if replyToID != "" {
		parent, err := s.store.GetMessage(ctx, replyToID)
		if err != nil {
			return Message{}, err
		}
		if parent.ConversationID != conversationID {
			return Message{}, ErrInvalidInput
		}
	}
	if attachmentID != "" {
		att, err := s.store.GetAttachment(ctx, attachmentID)
		if err != nil {
			return Message{}, err
		}
		if att.ConversationID != conversationID || att.UploaderID != actorID {
			return Message{}, ErrForbidden
		}
	}
	now := s.now().UTC()
	msg := Message{
		ID:              newID("msg"),
		ConversationID:  conversationID,
		SenderID:        actorID,
		Body:            text,
		ReplyToID:       strings.TrimSpace(replyToID),
		AttachmentID:    strings.TrimSpace(attachmentID),
		E2E:             e2e,
		ForwardedFromID: strings.TrimSpace(forwardedFrom),
		CreatedAt:       now,
	}
	if err := s.store.InsertMessage(ctx, msg); err != nil {
		return Message{}, err
	}
	_ = s.store.MarkRead(ctx, conversationID, actorID, now)
	s.publishMembers(ctx, conversationID, Event{
		Type: EventMessageCreated,
		Payload: map[string]any{
			"id":              msg.ID,
			"conversation_id": msg.ConversationID,
			"sender_id":       msg.SenderID,
			"body":            msg.Body,
			"reply_to_id":     msg.ReplyToID,
			"attachment_id":   msg.AttachmentID,
			"e2e":             msg.E2E,
			"created_at":      msg.CreatedAt.UTC().Format(time.RFC3339Nano),
		},
	})
	if !isBot && !e2e {
		go s.maybeDevBotReply(context.WithoutCancel(ctx), actorID, conversationID, text)
	}
	return msg, nil
}

func (s *Service) DeleteMessage(ctx context.Context, actorID, conversationID, messageID string) error {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return err
	}
	msg, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}
	if msg.ConversationID != conversationID {
		return ErrNotFound
	}
	if msg.SenderID != actorID {
		if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
			bot, botErr := s.store.GetBotByUserID(ctx, actorID)
			if botErr != nil {
				return ErrForbidden
			}
			grant, gerr := s.store.GetBotGrant(ctx, bot.ID, conversationID)
			if gerr != nil || !grant.CanManageMessages {
				return ErrForbidden
			}
		}
	}
	if err := s.store.SoftDeleteMessage(ctx, messageID, s.now().UTC()); err != nil {
		return err
	}
	s.publishMembers(ctx, conversationID, Event{
		Type: EventMessageDeleted,
		Payload: map[string]any{
			"id":              messageID,
			"conversation_id": conversationID,
		},
	})
	return nil
}

func (s *Service) MemberIDs(ctx context.Context, conversationID string) ([]string, error) {
	return s.store.MemberIDs(ctx, conversationID)
}

func (s *Service) requireMember(ctx context.Context, conversationID, userID string) error {
	ok, err := s.store.IsMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) requireReadable(ctx context.Context, conversationID, userID string) error {
	if err := s.requireMember(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.requireBotGrant(ctx, conversationID, userID, func(g BotGrant) bool { return g.CanRead })
}

func (s *Service) requireBotCanSend(ctx context.Context, conversationID, userID string) error {
	return s.requireBotGrant(ctx, conversationID, userID, func(g BotGrant) bool { return g.CanSend })
}

func (s *Service) requireBotCanManageMembers(ctx context.Context, conversationID, userID string) error {
	bot, err := s.store.GetBotByUserID(ctx, userID)
	if err != nil {
		return ErrForbidden
	}
	grant, err := s.store.GetBotGrant(ctx, bot.ID, conversationID)
	if err != nil || !grant.CanManageMembers {
		return ErrForbidden
	}
	return nil
}

func (s *Service) requireBotGrant(ctx context.Context, conversationID, userID string, ok func(BotGrant) bool) error {
	bot, err := s.store.GetBotByUserID(ctx, userID)
	if err != nil {
		return nil
	}
	grant, err := s.store.GetBotGrant(ctx, bot.ID, conversationID)
	if err != nil || !ok(grant) {
		return ErrForbidden
	}
	return nil
}

func (s *Service) summaryFor(ctx context.Context, actorID, conversationID string, peer UserRef) (ConversationSummary, error) {
	items, err := s.store.ListSummaries(ctx, actorID)
	if err != nil {
		return ConversationSummary{}, err
	}
	for _, item := range items {
		if item.ID == conversationID {
			if item.Peer.Username == "" {
				item.Peer = peer
			}
			return item, nil
		}
	}
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return ConversationSummary{}, err
	}
	kind := conv.Kind
	if kind == "" {
		kind = KindDirect
	}
	return ConversationSummary{
		ID:            conv.ID,
		Kind:          kind,
		Title:         conv.Title,
		Visibility:    conv.Visibility,
		Peer:          peer,
		CreatedAt:     conv.CreatedAt,
		LastMessageAt: conv.CreatedAt,
	}, nil
}

func (s *Service) publishMembers(ctx context.Context, conversationID string, event Event) {
	s.publishMembersExcept(ctx, conversationID, "", event)
}

func (s *Service) publishMembersExcept(ctx context.Context, conversationID, exceptID string, event Event) {
	if s.realtime == nil {
		return
	}
	ids, err := s.store.MemberIDs(ctx, conversationID)
	if err != nil {
		return
	}
	if exceptID != "" {
		filtered := make([]string, 0, len(ids))
		for _, id := range ids {
			if id != exceptID {
				filtered = append(filtered, id)
			}
		}
		ids = filtered
	}
	s.realtime.Publish(ids, event)
}
