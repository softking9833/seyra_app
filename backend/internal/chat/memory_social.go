package chat

import (
	"context"
	"time"
)

func (s *MemoryStore) IsBlocked(_ context.Context, actorID, otherID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, a := s.blocks[actorID][otherID]
	_, b := s.blocks[otherID][actorID]
	return a || b, nil
}

func (s *MemoryStore) BlockUser(_ context.Context, actorID, otherID string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocks[actorID] == nil {
		s.blocks[actorID] = map[string]struct{}{}
	}
	s.blocks[actorID][otherID] = struct{}{}
	return nil
}

func (s *MemoryStore) UnblockUser(_ context.Context, actorID, otherID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.blocks[actorID], otherID)
	return nil
}

func (s *MemoryStore) ListBlocked(_ context.Context, actorID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for id := range s.blocks[actorID] {
		out = append(out, id)
	}
	return out, nil
}

func (s *MemoryStore) ReportUser(_ context.Context, id, reporterID, targetID, reason string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports = append(s.reports, struct{ id, reporter, target, reason string }{id, reporterID, targetID, reason})
	return nil
}

func (s *MemoryStore) GetPrivacy(_ context.Context, userID string) (PrivacySettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.privacy[userID]; ok {
		return p, nil
	}
	return PrivacySettings{
		UserID: userID, LastSeenVisible: true, ReadReceipts: true,
		TypingVisible: true, ProfileVisible: true, NotificationPreview: true, PhotoVisible: true,
	}, nil
}

func (s *MemoryStore) PutPrivacy(_ context.Context, settings PrivacySettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.privacy[settings.UserID] = settings
	return nil
}

func (s *MemoryStore) PinMessage(_ context.Context, conversationID, messageID, _ string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range s.pins[conversationID] {
		if id == messageID {
			return nil
		}
	}
	s.pins[conversationID] = append(s.pins[conversationID], messageID)
	return nil
}

func (s *MemoryStore) UnpinMessage(_ context.Context, conversationID, messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var next []string
	for _, id := range s.pins[conversationID] {
		if id != messageID {
			next = append(next, id)
		}
	}
	s.pins[conversationID] = next
	return nil
}

func (s *MemoryStore) ListPins(_ context.Context, conversationID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.pins[conversationID]...), nil
}

func (s *MemoryStore) SetMutedConv(_ context.Context, userID, conversationID string, muted bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mutes[userID] == nil {
		s.mutes[userID] = map[string]bool{}
	}
	s.mutes[userID][conversationID] = muted
	return nil
}

func (s *MemoryStore) IsMutedConv(_ context.Context, userID, conversationID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mutes[userID][conversationID], nil
}

func (s *MemoryStore) SetArchived(_ context.Context, userID, conversationID string, archived bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.archives[userID] == nil {
		s.archives[userID] = map[string]bool{}
	}
	if archived {
		s.archives[userID][conversationID] = true
	} else {
		delete(s.archives[userID], conversationID)
	}
	return nil
}

func (s *MemoryStore) SaveDraft(_ context.Context, userID, conversationID, body string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.drafts[userID] == nil {
		s.drafts[userID] = map[string]string{}
	}
	s.drafts[userID][conversationID] = body
	return nil
}

func (s *MemoryStore) GetDraft(_ context.Context, userID, conversationID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.drafts[userID][conversationID], nil
}

func (s *MemoryStore) CreateInvite(_ context.Context, link InviteLink) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invites[link.Token] = link
	return nil
}

func (s *MemoryStore) GetInviteByToken(_ context.Context, token string) (InviteLink, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.invites[token]
	if !ok {
		return InviteLink{}, ErrNotFound
	}
	return link, nil
}

func (s *MemoryStore) RevokeInvite(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, link := range s.invites {
		if link.ID == id {
			link.RevokedAt = &at
			s.invites[token] = link
		}
	}
	return nil
}

func (s *MemoryStore) UpdateRoomMeta(_ context.Context, conversationID, title, description, visibility string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	conv, ok := s.conversations[conversationID]
	if !ok {
		return ErrNotFound
	}
	if title != "" {
		conv.Title = title
	}
	conv.Description = description
	if visibility != "" {
		conv.Visibility = visibility
	}
	s.conversations[conversationID] = conv
	return nil
}

func deviceKey(userID, deviceID string) string { return userID + "/" + deviceID }

func (s *MemoryStore) UpsertE2EDevice(_ context.Context, dev E2EDevice) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.e2eDevices[deviceKey(dev.UserID, dev.DeviceID)] = dev
	return nil
}

func (s *MemoryStore) RevokeE2EDevice(_ context.Context, userID, deviceID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := deviceKey(userID, deviceID)
	dev, ok := s.e2eDevices[key]
	if !ok {
		return ErrNotFound
	}
	dev.RevokedAt = &at
	s.e2eDevices[key] = dev
	return nil
}

func (s *MemoryStore) GetE2EDevice(_ context.Context, userID, deviceID string) (E2EDevice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dev, ok := s.e2eDevices[deviceKey(userID, deviceID)]
	if !ok || dev.RevokedAt != nil {
		return E2EDevice{}, ErrNotFound
	}
	return dev, nil
}

func (s *MemoryStore) ListE2EDevices(_ context.Context, userID string) ([]E2EDevice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []E2EDevice
	for _, dev := range s.e2eDevices {
		if dev.UserID == userID && dev.RevokedAt == nil {
			out = append(out, dev)
		}
	}
	return out, nil
}

func (s *MemoryStore) ReplacePreKeys(_ context.Context, userID, deviceID string, keys []E2EPreKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.e2ePrekeys[deviceKey(userID, deviceID)] = keys
	return nil
}

func (s *MemoryStore) ConsumePreKey(_ context.Context, userID, deviceID string) (E2EPreKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := deviceKey(userID, deviceID)
	keys := s.e2ePrekeys[key]
	if len(keys) == 0 {
		return E2EPreKey{}, ErrNotFound
	}
	pk := keys[0]
	s.e2ePrekeys[key] = keys[1:]
	return pk, nil
}

func (s *MemoryStore) InsertCall(_ context.Context, call CallSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[call.ID] = call
	return nil
}

func (s *MemoryStore) UpdateCallState(_ context.Context, id, state string, endedAt *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	call, ok := s.calls[id]
	if !ok {
		return ErrNotFound
	}
	call.State = state
	call.EndedAt = endedAt
	s.calls[id] = call
	return nil
}

func (s *MemoryStore) GetCall(_ context.Context, id string) (CallSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	call, ok := s.calls[id]
	if !ok {
		return CallSession{}, ErrNotFound
	}
	return call, nil
}

func (s *MemoryStore) ListCalls(_ context.Context, userID string, limit int) ([]CallSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []CallSession
	for _, call := range s.calls {
		if call.CallerID == userID {
			out = append(out, call)
			continue
		}
		for _, id := range call.ParticipantIDs {
			if id == userID {
				out = append(out, call)
				break
			}
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) DeleteCall(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.calls[id]; !ok {
		return ErrNotFound
	}
	delete(s.calls, id)
	return nil
}

func (s *MemoryStore) DeleteCallsForUser(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, call := range s.calls {
		keep := true
		if call.CallerID == userID {
			keep = false
		} else {
			for _, pid := range call.ParticipantIDs {
				if pid == userID {
					keep = false
					break
				}
			}
		}
		if !keep {
			delete(s.calls, id)
		}
	}
	return nil
}

func (s *MemoryStore) InsertBot(_ context.Context, bot Bot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bots[bot.ID] = bot
	s.botsByHash[bot.TokenHash] = bot.ID
	return nil
}

func (s *MemoryStore) GetBotByTokenHash(_ context.Context, hash string) (Bot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.botsByHash[hash]
	if !ok {
		return Bot{}, ErrNotFound
	}
	return s.bots[id], nil
}

func (s *MemoryStore) GetBotByID(_ context.Context, id string) (Bot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bot, ok := s.bots[id]
	if !ok {
		return Bot{}, ErrNotFound
	}
	return bot, nil
}

func (s *MemoryStore) GetBotByUserID(_ context.Context, userID string) (Bot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, bot := range s.bots {
		if bot.UserID == userID {
			return bot, nil
		}
	}
	return Bot{}, ErrNotFound
}

func (s *MemoryStore) SetMemberRestriction(_ context.Context, conversationID, userID string, canSend bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.restrictions == nil {
		s.restrictions = map[string]bool{}
	}
	s.restrictions[conversationID+"/"+userID] = canSend
	return nil
}

func (s *MemoryStore) MemberCanSend(_ context.Context, conversationID, userID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	can, ok := s.restrictions[conversationID+"/"+userID]
	if !ok {
		return true, nil
	}
	return can, nil
}

func (s *MemoryStore) ListBots(_ context.Context, ownerID string) ([]Bot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Bot
	for _, bot := range s.bots {
		if bot.OwnerID == ownerID {
			out = append(out, bot)
		}
	}
	return out, nil
}

func (s *MemoryStore) DeleteBot(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bot, ok := s.bots[id]
	if !ok {
		return ErrNotFound
	}
	delete(s.botsByHash, bot.TokenHash)
	delete(s.bots, id)
	return nil
}

func (s *MemoryStore) UpsertBotGrant(_ context.Context, grant BotGrant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.botGrants[grant.BotID+"/"+grant.ConversationID] = grant
	return nil
}

func (s *MemoryStore) GetBotGrant(_ context.Context, botID, conversationID string) (BotGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.botGrants[botID+"/"+conversationID]
	if !ok {
		return BotGrant{}, ErrNotFound
	}
	return g, nil
}

func (s *MemoryStore) TouchPresence(_ context.Context, userID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.presence[userID] = at
	return nil
}

func (s *MemoryStore) GetPresence(_ context.Context, userID string) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	at, ok := s.presence[userID]
	if !ok {
		return time.Time{}, ErrNotFound
	}
	return at, nil
}
