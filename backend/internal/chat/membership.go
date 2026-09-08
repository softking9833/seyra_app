package chat

import (
	"context"
	"errors"

	"seyra/backend/internal/auth"
)

func (s *Service) ListRoomMembers(ctx context.Context, actorID, conversationID string) ([]RoomMember, error) {
	if err := s.requireReadable(ctx, conversationID, actorID); err != nil {
		return nil, err
	}
	members, err := s.store.ListMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	out := make([]RoomMember, 0, len(members))
	for _, m := range members {
		user, err := s.directory.LookupID(ctx, m.UserID)
		if err != nil {
			user = UserRef{ID: m.UserID}
		}
		role := m.Role
		if role == "" {
			role = RoleMember
		}
		out = append(out, RoomMember{User: user, Role: role})
	}
	return out, nil
}

func (s *Service) AddMembers(ctx context.Context, actorID, conversationID string, usernames []string) ([]RoomMember, error) {
	conv, err := s.requireRoom(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
		if err2 := s.requireBotCanManageMembers(ctx, conversationID, actorID); err2 != nil {
			return nil, err
		}
	}
	existing, err := s.store.MemberIDs(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if len(existing)+len(usernames) > MaxRoomMembers {
		return nil, ErrInvalidInput
	}
	now := s.now().UTC()
	for _, raw := range usernames {
		normalized, err := auth.NormalizeUsername(raw)
		if err != nil {
			return nil, ErrInvalidInput
		}
		peer, err := s.directory.LookupUsername(ctx, normalized)
		if err != nil {
			return nil, err
		}
		if err := s.store.AddMember(ctx, conversationID, peer.ID, RoleMember, now); err != nil {
			if errors.Is(err, ErrAlreadyMember) {
				continue
			}
			return nil, err
		}
		s.publishMembers(ctx, conversationID, Event{
			Type: EventMemberJoined,
			Payload: map[string]any{
				"conversation_id": conversationID,
				"user_id":         peer.ID,
				"username":        peer.Username,
				"role":            RoleMember,
				"kind":            conv.Kind,
			},
		})
	}
	return s.ListRoomMembers(ctx, actorID, conversationID)
}

func (s *Service) RemoveMember(ctx context.Context, actorID, conversationID, targetID string) error {
	conv, err := s.requireRoom(ctx, conversationID)
	if err != nil {
		return err
	}
	targetRole, err := s.store.MemberRole(ctx, conversationID, targetID)
	if err != nil {
		return err
	}
	if targetRole == RoleOwner {
		return ErrOwnerProtected
	}
	if actorID == targetID {
		return s.LeaveRoom(ctx, actorID, conversationID)
	}
	actorRole, err := s.store.MemberRole(ctx, conversationID, actorID)
	if err != nil {
		return err
	}
	if !canManageMember(actorRole, targetRole) {
		if err := s.requireBotCanManageMembers(ctx, conversationID, actorID); err != nil || targetRole != RoleMember {
			return ErrForbidden
		}
	}
	others, err := s.store.MemberIDs(ctx, conversationID)
	if err != nil {
		return err
	}
	if err := s.store.RemoveMember(ctx, conversationID, targetID); err != nil {
		return err
	}
	s.publishTo(append([]string{targetID}, others...), Event{
		Type: EventMemberLeft,
		Payload: map[string]any{
			"conversation_id": conversationID,
			"user_id":         targetID,
			"kind":            conv.Kind,
		},
	})
	return nil
}

func (s *Service) LeaveRoom(ctx context.Context, actorID, conversationID string) error {
	conv, err := s.requireRoom(ctx, conversationID)
	if err != nil {
		return err
	}
	role, err := s.store.MemberRole(ctx, conversationID, actorID)
	if err != nil {
		return err
	}
	if role == RoleOwner {
		return ErrOwnerProtected
	}
	others, err := s.store.MemberIDs(ctx, conversationID)
	if err != nil {
		return err
	}
	if err := s.store.RemoveMember(ctx, conversationID, actorID); err != nil {
		return err
	}
	event := Event{
		Type: EventMemberLeft,
		Payload: map[string]any{
			"conversation_id": conversationID,
			"user_id":         actorID,
			"kind":            conv.Kind,
		},
	}
	s.publishTo(append(others, actorID), event)
	return nil
}

func (s *Service) SetMemberRole(ctx context.Context, actorID, conversationID, targetID, role string) error {
	conv, err := s.requireRoom(ctx, conversationID)
	if err != nil {
		return err
	}
	if role != RoleAdmin && role != RoleMember {
		return ErrInvalidInput
	}
	if err := s.requireOwner(ctx, conversationID, actorID); err != nil {
		return err
	}
	targetRole, err := s.store.MemberRole(ctx, conversationID, targetID)
	if err != nil {
		return err
	}
	if targetRole == RoleOwner {
		return ErrOwnerProtected
	}
	if actorID == targetID {
		return ErrOwnerProtected
	}
	if err := s.store.SetMemberRole(ctx, conversationID, targetID, role); err != nil {
		return err
	}
	s.publishMembers(ctx, conversationID, Event{
		Type: EventMemberUpdated,
		Payload: map[string]any{
			"conversation_id": conversationID,
			"user_id":         targetID,
			"role":            role,
			"kind":            conv.Kind,
		},
	})
	return nil
}

func (s *Service) requireRoom(ctx context.Context, conversationID string) (Conversation, error) {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return Conversation{}, err
	}
	if conv.Kind != KindGroup && conv.Kind != KindChannel {
		return Conversation{}, ErrInvalidInput
	}
	return conv, nil
}

func (s *Service) requireAdmin(ctx context.Context, conversationID, userID string) error {
	role, err := s.store.MemberRole(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if role != RoleOwner && role != RoleAdmin {
		return ErrForbidden
	}
	return nil
}

func (s *Service) requireOwner(ctx context.Context, conversationID, userID string) error {
	role, err := s.store.MemberRole(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if role != RoleOwner {
		return ErrForbidden
	}
	return nil
}

func canManageMember(actorRole, targetRole string) bool {
	if actorRole == RoleOwner {
		return targetRole != RoleOwner
	}
	if actorRole == RoleAdmin {
		return targetRole == RoleMember
	}
	return false
}

func (s *Service) publishTo(userIDs []string, event Event) {
	if s.realtime == nil || len(userIDs) == 0 {
		return
	}
	s.realtime.Publish(userIDs, event)
}
