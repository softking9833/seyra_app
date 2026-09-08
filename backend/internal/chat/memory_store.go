package chat

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu            sync.Mutex
	conversations map[string]Conversation
	byPair        map[string]string
	members       map[string]map[string]Member
	messages      map[string]Message
	attachments   map[string]Attachment
	reactions     map[string]map[string]map[string]struct{}
	order         map[string][]string
	blocks        map[string]map[string]struct{}
	privacy       map[string]PrivacySettings
	pins          map[string][]string
	mutes         map[string]map[string]bool
	archives      map[string]map[string]bool
	drafts        map[string]map[string]string
	invites       map[string]InviteLink
	e2eDevices    map[string]E2EDevice
	e2ePrekeys    map[string][]E2EPreKey
	calls         map[string]CallSession
	bots          map[string]Bot
	botsByHash    map[string]string
	botGrants     map[string]BotGrant
	restrictions  map[string]bool
	presence      map[string]time.Time
	reports       []struct{ id, reporter, target, reason string }
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: map[string]Conversation{},
		byPair:        map[string]string{},
		members:       map[string]map[string]Member{},
		messages:      map[string]Message{},
		attachments:   map[string]Attachment{},
		reactions:     map[string]map[string]map[string]struct{}{},
		order:         map[string][]string{},
		blocks:        map[string]map[string]struct{}{},
		privacy:       map[string]PrivacySettings{},
		pins:          map[string][]string{},
		mutes:         map[string]map[string]bool{},
		archives:      map[string]map[string]bool{},
		drafts:        map[string]map[string]string{},
		invites:       map[string]InviteLink{},
		e2eDevices:    map[string]E2EDevice{},
		e2ePrekeys:    map[string][]E2EPreKey{},
		calls:         map[string]CallSession{},
		bots:          map[string]Bot{},
		botsByHash:    map[string]string{},
		botGrants:     map[string]BotGrant{},
		restrictions:  map[string]bool{},
		presence:      map[string]time.Time{},
	}
}

func (s *MemoryStore) CreateConversation(_ context.Context, conv Conversation, memberIDs [2]string) error {
	if conv.Kind == "" {
		conv.Kind = KindDirect
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byPair[conv.PairKey]; ok {
		return nil
	}
	s.conversations[conv.ID] = conv
	s.byPair[conv.PairKey] = conv.ID
	s.members[conv.ID] = map[string]Member{}
	now := conv.CreatedAt
	for _, id := range memberIDs {
		s.members[conv.ID][id] = Member{
			ConversationID: conv.ID,
			UserID:         id,
			Role:           RoleMember,
			CreatedAt:      now,
		}
	}
	return nil
}

func (s *MemoryStore) CreateRoom(_ context.Context, conv Conversation, ownerID string, memberIDs []string) error {
	if conv.Visibility == "" {
		conv.Visibility = VisibilityPrivate
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversations[conv.ID] = conv
	s.members[conv.ID] = map[string]Member{}
	now := conv.CreatedAt
	for _, id := range memberIDs {
		role := RoleMember
		if id == ownerID {
			role = RoleOwner
		}
		s.members[conv.ID][id] = Member{
			ConversationID: conv.ID,
			UserID:         id,
			Role:           role,
			CreatedAt:      now,
		}
	}
	return nil
}

func (s *MemoryStore) GetConversationByPairKey(_ context.Context, key string) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byPair[key]
	if !ok {
		return Conversation{}, ErrNotFound
	}
	return s.conversations[id], nil
}

func (s *MemoryStore) GetConversation(_ context.Context, id string) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conv, ok := s.conversations[id]
	if !ok {
		return Conversation{}, ErrNotFound
	}
	return conv, nil
}

func (s *MemoryStore) IsMember(_ context.Context, conversationID, userID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.members[conversationID]
	if !ok {
		return false, ErrNotFound
	}
	_, member := members[userID]
	return member, nil
}

func (s *MemoryStore) MemberRole(_ context.Context, conversationID, userID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.members[conversationID]
	if !ok {
		return "", ErrNotFound
	}
	me, ok := members[userID]
	if !ok {
		return "", ErrForbidden
	}
	if me.Role == "" {
		return RoleMember, nil
	}
	return me.Role, nil
}

func (s *MemoryStore) ListMembers(_ context.Context, conversationID string) ([]Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.members[conversationID]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]Member, 0, len(members))
	for _, m := range members {
		if m.Role == "" {
			m.Role = RoleMember
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *MemoryStore) AddMember(_ context.Context, conversationID, userID, role string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.members[conversationID]
	if !ok {
		return ErrNotFound
	}
	if _, exists := members[userID]; exists {
		return ErrAlreadyMember
	}
	if role == "" {
		role = RoleMember
	}
	members[userID] = Member{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		CreatedAt:      at,
	}
	return nil
}

func (s *MemoryStore) RemoveMember(_ context.Context, conversationID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.members[conversationID]
	if !ok {
		return ErrNotFound
	}
	if _, exists := members[userID]; !exists {
		return ErrNotFound
	}
	delete(members, userID)
	return nil
}

func (s *MemoryStore) SetMemberRole(_ context.Context, conversationID, userID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.members[conversationID]
	if !ok {
		return ErrNotFound
	}
	me, exists := members[userID]
	if !exists {
		return ErrNotFound
	}
	me.Role = role
	members[userID] = me
	return nil
}

func (s *MemoryStore) MemberIDs(_ context.Context, conversationID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.members[conversationID]
	if !ok {
		return nil, ErrNotFound
	}
	ids := make([]string, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *MemoryStore) ListSummaries(_ context.Context, userID string) ([]ConversationSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ConversationSummary
	for convID, members := range s.members {
		me, ok := members[userID]
		if !ok {
			continue
		}
		conv := s.conversations[convID]
		kind := conv.Kind
		if kind == "" {
			kind = KindDirect
		}
		var peerID string
		if kind == KindDirect {
			for id := range members {
				if id != userID {
					peerID = id
					break
				}
			}
		}
		summary := ConversationSummary{
			ID:          conv.ID,
			Kind:        kind,
			Title:       conv.Title,
			Visibility:  conv.Visibility,
			MemberCount: len(members),
			Peer:        UserRef{ID: peerID},
			CreatedAt:   conv.CreatedAt,
		}
		ids := s.order[convID]
		for i := len(ids) - 1; i >= 0; i-- {
			msg := s.messages[ids[i]]
			if msg.DeletedAt != nil {
				continue
			}
			if msg.E2E {
				summary.LastMessagePreview = "Encrypted message"
			} else {
				summary.LastMessagePreview = msg.Body
			}
			summary.LastMessageAt = msg.CreatedAt
			break
		}
		if summary.LastMessageAt.IsZero() {
			summary.LastMessageAt = conv.CreatedAt
		}
		unread := 0
		for _, id := range ids {
			msg := s.messages[id]
			if msg.DeletedAt != nil || msg.SenderID == userID {
				continue
			}
			if me.LastReadAt == nil || msg.CreatedAt.After(*me.LastReadAt) {
				unread++
			}
		}
		summary.UnreadCount = unread
		out = append(out, summary)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].LastMessageAt.After(out[i].LastMessageAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) InsertMessage(_ context.Context, message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conversations[message.ConversationID]; !ok {
		return ErrNotFound
	}
	s.messages[message.ID] = message
	s.order[message.ConversationID] = append(s.order[message.ConversationID], message.ID)
	return nil
}

func (s *MemoryStore) ListMessages(_ context.Context, conversationID, beforeID string, limit int) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := s.order[conversationID]
	end := len(ids)
	if beforeID != "" {
		end = -1
		for i, id := range ids {
			if id == beforeID {
				end = i
				break
			}
		}
		if end < 0 {
			return nil, ErrNotFound
		}
	}
	var newestFirst []Message
	for i := end - 1; i >= 0 && len(newestFirst) < limit; i-- {
		msg := s.messages[ids[i]]
		if msg.DeletedAt != nil {
			continue
		}
		newestFirst = append(newestFirst, msg)
	}
	out := make([]Message, 0, len(newestFirst))
	for i := len(newestFirst) - 1; i >= 0; i-- {
		out = append(out, newestFirst[i])
	}
	return out, nil
}

func (s *MemoryStore) GetMessage(_ context.Context, id string) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg, ok := s.messages[id]
	if !ok {
		return Message{}, ErrNotFound
	}
	return msg, nil
}

func (s *MemoryStore) SoftDeleteMessage(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg, ok := s.messages[id]
	if !ok {
		return ErrNotFound
	}
	msg.DeletedAt = &at
	s.messages[id] = msg
	return nil
}

func (s *MemoryStore) MarkRead(_ context.Context, conversationID, userID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.members[conversationID]
	if !ok {
		return ErrNotFound
	}
	me, ok := members[userID]
	if !ok {
		return ErrForbidden
	}
	me.LastReadAt = &at
	members[userID] = me
	return nil
}

func (s *MemoryStore) UpdateMessageBody(_ context.Context, id, body string, editedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg, ok := s.messages[id]
	if !ok {
		return ErrNotFound
	}
	msg.Body = body
	msg.EditedAt = &editedAt
	s.messages[id] = msg
	return nil
}

func (s *MemoryStore) InsertAttachment(_ context.Context, att Attachment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attachments[att.ID] = att
	return nil
}

func (s *MemoryStore) GetAttachment(_ context.Context, id string) (Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	att, ok := s.attachments[id]
	if !ok {
		return Attachment{}, ErrNotFound
	}
	return att, nil
}

func (s *MemoryStore) ListAttachments(_ context.Context, conversationID string, limit int) ([]Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Attachment
	for _, att := range s.attachments {
		if att.ConversationID == conversationID {
			out = append(out, att)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) DeleteAttachment(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.attachments[id]; !ok {
		return ErrNotFound
	}
	delete(s.attachments, id)
	return nil
}

func (s *MemoryStore) SearchMessages(_ context.Context, userID, query string, limit int) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	needle := strings.ToLower(query)
	var out []Message
	for convID, members := range s.members {
		if _, ok := members[userID]; !ok {
			continue
		}
		for _, id := range s.order[convID] {
			msg := s.messages[id]
			if msg.DeletedAt != nil {
				continue
			}
			if !strings.Contains(strings.ToLower(msg.Body), needle) {
				continue
			}
			out = append(out, msg)
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) SearchConversations(_ context.Context, userID, query string, limit int) ([]ConversationSummary, error) {
	items, err := s.ListSummaries(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(query)
	var out []ConversationSummary
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Title), needle) || strings.Contains(strings.ToLower(item.Peer.Username), needle) {
			out = append(out, item)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (s *MemoryStore) DiscoverChannels(_ context.Context, query string, limit int) ([]ConversationSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	needle := strings.ToLower(strings.TrimSpace(query))
	var out []ConversationSummary
	for _, conv := range s.conversations {
		if conv.Kind != KindChannel || conv.Visibility != VisibilityPublic {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(conv.Title), needle) {
			continue
		}
		out = append(out, ConversationSummary{
			ID:         conv.ID,
			Kind:       conv.Kind,
			Title:      conv.Title,
			Visibility: conv.Visibility,
			CreatedAt:  conv.CreatedAt,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *MemoryStore) SetVisibility(_ context.Context, conversationID, visibility string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	conv, ok := s.conversations[conversationID]
	if !ok {
		return ErrNotFound
	}
	conv.Visibility = visibility
	s.conversations[conversationID] = conv
	return nil
}

func (s *MemoryStore) ToggleReaction(_ context.Context, messageID, userID, emoji string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.messages[messageID]; !ok {
		return ErrNotFound
	}
	if s.reactions[messageID] == nil {
		s.reactions[messageID] = map[string]map[string]struct{}{}
	}
	if s.reactions[messageID][userID] == nil {
		s.reactions[messageID][userID] = map[string]struct{}{}
	}
	if _, ok := s.reactions[messageID][userID][emoji]; ok {
		delete(s.reactions[messageID][userID], emoji)
		return nil
	}
	s.reactions[messageID][userID][emoji] = struct{}{}
	return nil
}
