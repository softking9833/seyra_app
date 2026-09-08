package chat

import (
	"context"
	"strings"
	"unicode/utf8"
)

func (s *Service) React(ctx context.Context, actorID, conversationID, messageID, emoji string) error {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return err
	}
	if err := s.requireBotCanSend(ctx, conversationID, actorID); err != nil {
		return err
	}
	face := strings.TrimSpace(emoji)
	if face == "" || utf8.RuneCountInString(face) > 8 {
		return ErrInvalidInput
	}
	msg, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}
	if msg.ConversationID != conversationID {
		return ErrNotFound
	}
	if err := s.store.ToggleReaction(ctx, messageID, actorID, face, s.now().UTC()); err != nil {
		return err
	}
	s.publishMembers(ctx, conversationID, Event{
		Type: EventMessageUpdated,
		Payload: map[string]any{
			"id":              messageID,
			"conversation_id": conversationID,
			"reaction":        face,
			"user_id":         actorID,
		},
	})
	return nil
}
