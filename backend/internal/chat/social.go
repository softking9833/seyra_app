package chat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"seyra/backend/internal/auth"
)

func (s *Service) Block(ctx context.Context, actorID, username string) error {
	peer, err := s.lookupUser(ctx, username)
	if err != nil {
		return err
	}
	return s.store.BlockUser(ctx, actorID, peer.ID, s.now().UTC())
}

func (s *Service) Unblock(ctx context.Context, actorID, userID string) error {
	return s.store.UnblockUser(ctx, actorID, userID)
}

func (s *Service) ListBlocked(ctx context.Context, actorID string) ([]UserRef, error) {
	ids, err := s.store.ListBlocked(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]UserRef, 0, len(ids))
	for _, id := range ids {
		u, err := s.directory.LookupID(ctx, id)
		if err != nil {
			u = UserRef{ID: id}
		}
		out = append(out, u)
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, actorID, username, reason string) error {
	peer, err := s.lookupUser(ctx, username)
	if err != nil {
		return err
	}
	r := strings.TrimSpace(reason)
	if r == "" || len(r) > 500 {
		return ErrInvalidInput
	}
	return s.store.ReportUser(ctx, newID("rpt"), actorID, peer.ID, r, s.now().UTC())
}

func (s *Service) GetPrivacy(ctx context.Context, actorID string) (PrivacySettings, error) {
	return s.store.GetPrivacy(ctx, actorID)
}

func (s *Service) PutPrivacy(ctx context.Context, actorID string, in PrivacySettings) error {
	in.UserID = actorID
	return s.store.PutPrivacy(ctx, in)
}

func (s *Service) Pin(ctx context.Context, actorID, conversationID, messageID string) error {
	if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
		if err2 := s.requireMember(ctx, conversationID, actorID); err2 != nil {
			return err2
		}
		conv, err := s.store.GetConversation(ctx, conversationID)
		if err != nil {
			return err
		}
		if conv.Kind != KindDirect {
			return ErrForbidden
		}
	}
	msg, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}
	if msg.ConversationID != conversationID {
		return ErrNotFound
	}
	if err := s.store.PinMessage(ctx, conversationID, messageID, actorID, s.now().UTC()); err != nil {
		return err
	}
	s.publishMembers(ctx, conversationID, Event{
		Type:    EventPinned,
		Payload: map[string]any{"conversation_id": conversationID, "message_id": messageID},
	})
	return nil
}

func (s *Service) Unpin(ctx context.Context, actorID, conversationID, messageID string) error {
	if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
		conv, err := s.store.GetConversation(ctx, conversationID)
		if err != nil {
			return err
		}
		if conv.Kind != KindDirect {
			return ErrForbidden
		}
		if err := s.requireMember(ctx, conversationID, actorID); err != nil {
			return err
		}
	}
	if err := s.store.UnpinMessage(ctx, conversationID, messageID); err != nil {
		return err
	}
	s.publishMembers(ctx, conversationID, Event{
		Type:    EventUnpinned,
		Payload: map[string]any{"conversation_id": conversationID, "message_id": messageID},
	})
	return nil
}

func (s *Service) ListPinned(ctx context.Context, actorID, conversationID string) ([]Message, error) {
	if err := s.requireReadable(ctx, conversationID, actorID); err != nil {
		return nil, err
	}
	ids, err := s.store.ListPins(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	var out []Message
	for _, id := range ids {
		msg, err := s.store.GetMessage(ctx, id)
		if err != nil {
			continue
		}
		out = append(out, msg)
	}
	return out, nil
}

func (s *Service) SetMuted(ctx context.Context, actorID, conversationID string, muted bool) error {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return err
	}
	return s.store.SetMutedConv(ctx, actorID, conversationID, muted)
}

func (s *Service) SetArchived(ctx context.Context, actorID, conversationID string, archived bool) error {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return err
	}
	return s.store.SetArchived(ctx, actorID, conversationID, archived)
}

func (s *Service) SaveDraft(ctx context.Context, actorID, conversationID, body string) error {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return err
	}
	return s.store.SaveDraft(ctx, actorID, conversationID, body, s.now().UTC())
}

func (s *Service) GetDraft(ctx context.Context, actorID, conversationID string) (string, error) {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return "", err
	}
	return s.store.GetDraft(ctx, actorID, conversationID)
}

func (s *Service) CreateInvite(ctx context.Context, actorID, conversationID string) (InviteLink, error) {
	if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
		return InviteLink{}, err
	}
	token := newID("inv")
	raw := make([]byte, 18)
	_, _ = rand.Read(raw)
	token = hex.EncodeToString(raw)
	link := InviteLink{
		ID: newID("lnk"), ConversationID: conversationID, Token: token,
		CreatedBy: actorID, CreatedAt: s.now().UTC(),
	}
	if err := s.store.CreateInvite(ctx, link); err != nil {
		return InviteLink{}, err
	}
	return link, nil
}

func (s *Service) JoinInvite(ctx context.Context, actorID, token string) (ConversationSummary, error) {
	link, err := s.store.GetInviteByToken(ctx, token)
	if err != nil {
		return ConversationSummary{}, err
	}
	if link.RevokedAt != nil {
		return ConversationSummary{}, ErrForbidden
	}
	if blocked, err := s.store.IsBlocked(ctx, actorID, link.CreatedBy); err != nil {
		return ConversationSummary{}, err
	} else if blocked {
		return ConversationSummary{}, ErrForbidden
	}
	if err := s.store.AddMember(ctx, link.ConversationID, actorID, RoleMember, s.now().UTC()); err != nil && !errors.Is(err, ErrAlreadyMember) {
		return ConversationSummary{}, err
	}
	return s.summaryFor(ctx, actorID, link.ConversationID, UserRef{})
}

func (s *Service) UpdateRoom(ctx context.Context, actorID, conversationID, title, description, visibility string) error {
	if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
		return err
	}
	return s.store.UpdateRoomMeta(ctx, conversationID, title, description, visibility)
}

func (s *Service) TransferOwnership(ctx context.Context, actorID, conversationID, targetID string) error {
	if err := s.requireOwner(ctx, conversationID, actorID); err != nil {
		return err
	}
	if _, err := s.store.MemberRole(ctx, conversationID, targetID); err != nil {
		return err
	}
	if err := s.store.SetMemberRole(ctx, conversationID, targetID, RoleOwner); err != nil {
		return err
	}
	return s.store.SetMemberRole(ctx, conversationID, actorID, RoleAdmin)
}

func (s *Service) ForwardMessage(ctx context.Context, actorID, sourceConv, messageID, destConv string) (Message, error) {
	if err := s.requireMember(ctx, sourceConv, actorID); err != nil {
		return Message{}, err
	}
	msg, err := s.store.GetMessage(ctx, messageID)
	if err != nil {
		return Message{}, err
	}
	if msg.ConversationID != sourceConv {
		return Message{}, ErrNotFound
	}
	if msg.E2E {
		return Message{}, ErrForbidden
	}
	if strings.TrimSpace(msg.AttachmentID) != "" {
		return Message{}, ErrForbidden
	}
	copied, err := s.sendMessage(ctx, actorID, destConv, msg.Body, "", "", false, messageID)
	if err != nil {
		return Message{}, err
	}
	return copied, nil
}

func (s *Service) PublishKeys(ctx context.Context, actorID string, dev E2EDevice, prekeys []E2EPreKey) error {
	dev.UserID = actorID
	dev.CreatedAt = s.now().UTC()
	if dev.DeviceID == "" || dev.IdentityPublic == "" {
		return ErrInvalidInput
	}
	if err := s.store.UpsertE2EDevice(ctx, dev); err != nil {
		return err
	}
	for i := range prekeys {
		prekeys[i].UserID = actorID
		prekeys[i].DeviceID = dev.DeviceID
	}
	return s.store.ReplacePreKeys(ctx, actorID, dev.DeviceID, prekeys)
}

func (s *Service) FetchBundle(ctx context.Context, actorID, userID, deviceID string) (E2EDevice, E2EPreKey, error) {
	if actorID == "" || userID == "" {
		return E2EDevice{}, E2EPreKey{}, ErrForbidden
	}
	if actorID != userID {
		blocked, err := s.store.IsBlocked(ctx, actorID, userID)
		if err != nil {
			return E2EDevice{}, E2EPreKey{}, err
		}
		if blocked {
			return E2EDevice{}, E2EPreKey{}, ErrForbidden
		}
		if _, err := s.store.GetConversationByPairKey(ctx, pairKey(actorID, userID)); err != nil {
			return E2EDevice{}, E2EPreKey{}, ErrForbidden
		}
	}
	dev, err := s.store.GetE2EDevice(ctx, userID, deviceID)
	if err != nil {
		devs, err2 := s.store.ListE2EDevices(ctx, userID)
		if err2 != nil || len(devs) == 0 {
			return E2EDevice{}, E2EPreKey{}, ErrNotFound
		}
		dev = devs[0]
	}
	if actorID == userID {
		return dev, E2EPreKey{}, nil
	}
	pk, err := s.store.ConsumePreKey(ctx, dev.UserID, dev.DeviceID)
	if err != nil {
		return dev, E2EPreKey{}, nil
	}
	return dev, pk, nil
}

func (s *Service) lookupUser(ctx context.Context, username string) (UserRef, error) {
	normalized, err := auth.NormalizeUsername(username)
	if err != nil {
		return UserRef{}, ErrInvalidInput
	}
	return s.directory.LookupUsername(ctx, normalized)
}

func HashBotToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) CreateBot(ctx context.Context, ownerID, botUserID, username string) (Bot, string, error) {
	normalized, err := auth.NormalizeUsername(username)
	if err != nil {
		return Bot{}, "", ErrInvalidInput
	}
	raw := make([]byte, 24)
	_, _ = rand.Read(raw)
	token := "bot_" + hex.EncodeToString(raw)
	bot := Bot{
		ID: newID("bot"), OwnerID: ownerID, UserID: botUserID,
		Username: normalized, TokenHash: HashBotToken(token), CreatedAt: s.now().UTC(),
	}
	if err := s.store.InsertBot(ctx, bot); err != nil {
		return Bot{}, "", err
	}
	return bot, token, nil
}

func (s *Service) GrantBot(ctx context.Context, actorID, botID, conversationID string, grant BotGrant) error {
	if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
		return err
	}
	bot, err := s.store.GetBotByID(ctx, botID)
	if err != nil {
		return err
	}
	if err := s.store.AddMember(ctx, conversationID, bot.UserID, RoleMember, s.now().UTC()); err != nil && !errors.Is(err, ErrAlreadyMember) {
		return err
	}
	grant.BotID = botID
	grant.ConversationID = conversationID
	return s.store.UpsertBotGrant(ctx, grant)
}

func (s *Service) BotByToken(ctx context.Context, token string) (Bot, error) {
	return s.store.GetBotByTokenHash(ctx, HashBotToken(token))
}

func (s *Service) StartCall(ctx context.Context, actorID, conversationID, kind string, payload map[string]any) (CallSession, error) {
	if err := s.requireMember(ctx, conversationID, actorID); err != nil {
		return CallSession{}, err
	}
	if kind != "voice" && kind != "video" {
		return CallSession{}, ErrInvalidInput
	}
	ids, err := s.store.MemberIDs(ctx, conversationID)
	if err != nil {
		return CallSession{}, err
	}
	call := CallSession{
		ID: newID("call"), ConversationID: conversationID, CallerID: actorID,
		Kind: kind, State: "ringing", CreatedAt: s.now().UTC(), ParticipantIDs: ids,
	}
	if err := s.store.InsertCall(ctx, call); err != nil {
		return CallSession{}, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["call_id"] = call.ID
	payload["conversation_id"] = conversationID
	payload["caller_id"] = actorID
	payload["kind"] = kind
	payload["action"] = "offer"
	if caller, err := s.directory.LookupID(ctx, actorID); err == nil {
		payload["username"] = caller.Username
	}
	s.publishMembersExcept(ctx, conversationID, actorID, Event{Type: EventCallSignal, Payload: payload})
	return call, nil
}

func (s *Service) SignalCall(ctx context.Context, actorID, callID, action string, payload map[string]any) error {
	call, err := s.store.GetCall(ctx, callID)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, call.ConversationID, actorID); err != nil {
		return err
	}
	switch call.State {
	case "ended", "rejected", "missed":
		if action == "ice" || action == "answer" {
			return ErrForbidden
		}
	}
	now := s.now().UTC()
	switch action {
	case "answer":
		_ = s.store.UpdateCallState(ctx, callID, "active", nil)
	case "reject":
		_ = s.store.UpdateCallState(ctx, callID, "rejected", &now)
	case "hangup":
		_ = s.store.UpdateCallState(ctx, callID, "ended", &now)
	case "missed":
		_ = s.store.UpdateCallState(ctx, callID, "missed", &now)
	case "ice":
	default:
		return ErrInvalidInput
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["call_id"] = callID
	payload["conversation_id"] = call.ConversationID
	payload["from_id"] = actorID
	payload["action"] = action
	s.publishMembers(ctx, call.ConversationID, Event{Type: EventCallSignal, Payload: payload})
	return nil
}

func (s *Service) ListCalls(ctx context.Context, actorID string) ([]CallSession, error) {
	return s.store.ListCalls(ctx, actorID, 50)
}

func (s *Service) CallPeerName(ctx context.Context, actorID, conversationID string) string {
	conv, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return "Unknown"
	}
	if conv.Kind == KindDirect {
		ids, err := s.store.MemberIDs(ctx, conversationID)
		if err == nil {
			for _, id := range ids {
				if id == actorID {
					continue
				}
				ref, lookupErr := s.directory.LookupID(ctx, id)
				if lookupErr == nil && ref.Username != "" {
					return ref.Username
				}
			}
		}
	}
	if strings.TrimSpace(conv.Title) != "" {
		return conv.Title
	}
	return "Unknown"
}

func (s *Service) DeleteCall(ctx context.Context, actorID, callID string) error {
	call, err := s.store.GetCall(ctx, callID)
	if err != nil {
		return err
	}
	allowed := call.CallerID == actorID
	if !allowed {
		for _, id := range call.ParticipantIDs {
			if id == actorID {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		return ErrForbidden
	}
	return s.store.DeleteCall(ctx, callID)
}

func (s *Service) ClearCallHistory(ctx context.Context, actorID string) error {
	return s.store.DeleteCallsForUser(ctx, actorID)
}

func (s *Service) HandleRealtime(ctx context.Context, actorID string, event Event) error {
	switch event.Type {
	case EventTyping:
		conv, _ := event.Payload["conversation_id"].(string)
		typing, _ := event.Payload["typing"].(bool)
		if conv == "" {
			return ErrInvalidInput
		}
		return s.PublishTyping(ctx, actorID, conv, typing)
	case EventCallSignal:
	default:
		return nil
	}
	callID, _ := event.Payload["call_id"].(string)
	action, _ := event.Payload["action"].(string)
	if callID == "" {
		if action != "" && action != "offer" {
			return ErrInvalidInput
		}
		kind, _ := event.Payload["kind"].(string)
		conv, _ := event.Payload["conversation_id"].(string)
		if conv == "" || (kind != "voice" && kind != "video") {
			return ErrInvalidInput
		}
		_, err := s.StartCall(ctx, actorID, conv, kind, event.Payload)
		return err
	}
	if action == "" {
		return ErrInvalidInput
	}
	return s.SignalCall(ctx, actorID, callID, action, event.Payload)
}

func (s *Service) ListBots(ctx context.Context, ownerID string) ([]Bot, error) {
	return s.store.ListBots(ctx, ownerID)
}

func (s *Service) DeleteBot(ctx context.Context, actorID, botID string) (Bot, error) {
	bot, err := s.store.GetBotByID(ctx, botID)
	if err != nil {
		return Bot{}, err
	}
	if bot.OwnerID != actorID {
		return Bot{}, ErrForbidden
	}
	if err := s.store.DeleteBot(ctx, botID); err != nil {
		return Bot{}, err
	}
	return bot, nil
}

func (s *Service) RevokeInvite(ctx context.Context, actorID, conversationID, token string) error {
	if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
		return err
	}
	link, err := s.store.GetInviteByToken(ctx, token)
	if err != nil {
		return err
	}
	if link.ConversationID != conversationID {
		return ErrNotFound
	}
	return s.store.RevokeInvite(ctx, link.ID, s.now().UTC())
}

func (s *Service) RestrictMember(ctx context.Context, actorID, conversationID, userID string, canSend bool) error {
	if err := s.requireAdmin(ctx, conversationID, actorID); err != nil {
		return err
	}
	role, err := s.store.MemberRole(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if role == RoleOwner {
		return ErrOwnerProtected
	}
	return s.store.SetMemberRestriction(ctx, conversationID, userID, canSend)
}

func (s *Service) RevokeE2EDevice(ctx context.Context, actorID, deviceID string) error {
	return s.store.RevokeE2EDevice(ctx, actorID, deviceID, s.now().UTC())
}

func (s *Service) ListE2EDevices(ctx context.Context, actorID string) ([]E2EDevice, error) {
	return s.store.ListE2EDevices(ctx, actorID)
}

func (s *Service) maybeDevBotReply(ctx context.Context, actorID, conversationID, body string) {
	fields := strings.Fields(strings.TrimSpace(body))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return
	}
	cmd := strings.ToLower(fields[0])
	switch cmd {
	case "/ping", "/help", "/whoami":
	default:
		return
	}
	members, err := s.store.ListMembers(ctx, conversationID)
	if err != nil {
		return
	}
	for _, member := range members {
		if member.UserID == actorID {
			continue
		}
		bot, err := s.store.GetBotByUserID(ctx, member.UserID)
		if err != nil {
			continue
		}
		grant, err := s.store.GetBotGrant(ctx, bot.ID, conversationID)
		if err != nil || !grant.CanSend {
			continue
		}
		reply := "pong"
		switch cmd {
		case "/help":
			reply = "Commands: /ping, /help, /whoami"
		case "/whoami":
			reply = "bot:" + bot.Username
		}
		_, _ = s.SendMessage(ctx, bot.UserID, conversationID, reply, "", "", false)
	}
}
