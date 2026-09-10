package chat

import (
	"context"
	"time"
)

type Directory interface {
	LookupUsername(ctx context.Context, username string) (UserRef, error)
	LookupID(ctx context.Context, id string) (UserRef, error)
}

type Store interface {
	CreateConversation(ctx context.Context, conv Conversation, memberIDs [2]string) error
	CreateRoom(ctx context.Context, conv Conversation, ownerID string, memberIDs []string) error
	GetConversationByPairKey(ctx context.Context, pairKey string) (Conversation, error)
	GetConversation(ctx context.Context, id string) (Conversation, error)
	IsMember(ctx context.Context, conversationID, userID string) (bool, error)
	MemberRole(ctx context.Context, conversationID, userID string) (string, error)
	MemberIDs(ctx context.Context, conversationID string) ([]string, error)
	ListMembers(ctx context.Context, conversationID string) ([]Member, error)
	AddMember(ctx context.Context, conversationID, userID, role string, at time.Time) error
	RemoveMember(ctx context.Context, conversationID, userID string) error
	SetMemberRole(ctx context.Context, conversationID, userID, role string) error
	ListSummaries(ctx context.Context, userID string) ([]ConversationSummary, error)
	InsertMessage(ctx context.Context, message Message) error
	UpdateMessageBody(ctx context.Context, id, body string, editedAt time.Time) error
	ListMessages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error)
	GetMessage(ctx context.Context, id string) (Message, error)
	SoftDeleteMessage(ctx context.Context, id string, at time.Time) error
	MarkRead(ctx context.Context, conversationID, userID string, at time.Time) error
	InsertAttachment(ctx context.Context, att Attachment) error
	GetAttachment(ctx context.Context, id string) (Attachment, error)
	ListAttachments(ctx context.Context, conversationID string, limit int) ([]Attachment, error)
	DeleteAttachment(ctx context.Context, id string) error
	SearchMessages(ctx context.Context, userID, query string, limit int) ([]Message, error)
	SearchConversations(ctx context.Context, userID, query string, limit int) ([]ConversationSummary, error)
	DiscoverChannels(ctx context.Context, query string, limit int) ([]ConversationSummary, error)
	SetVisibility(ctx context.Context, conversationID, visibility string) error
	ToggleReaction(ctx context.Context, messageID, userID, emoji string, at time.Time) error
	IsBlocked(ctx context.Context, actorID, otherID string) (bool, error)
	BlockUser(ctx context.Context, actorID, otherID string, at time.Time) error
	UnblockUser(ctx context.Context, actorID, otherID string) error
	ListBlocked(ctx context.Context, actorID string) ([]string, error)
	ReportUser(ctx context.Context, id, reporterID, targetID, reason string, at time.Time) error
	GetPrivacy(ctx context.Context, userID string) (PrivacySettings, error)
	PutPrivacy(ctx context.Context, settings PrivacySettings) error
	PinMessage(ctx context.Context, conversationID, messageID, actorID string, at time.Time) error
	UnpinMessage(ctx context.Context, conversationID, messageID string) error
	ListPins(ctx context.Context, conversationID string) ([]string, error)
	SetMutedConv(ctx context.Context, userID, conversationID string, muted bool) error
	IsMutedConv(ctx context.Context, userID, conversationID string) (bool, error)
	SetArchived(ctx context.Context, userID, conversationID string, archived bool) error
	SaveDraft(ctx context.Context, userID, conversationID, body string, at time.Time) error
	GetDraft(ctx context.Context, userID, conversationID string) (string, error)
	CreateInvite(ctx context.Context, link InviteLink) error
	GetInviteByToken(ctx context.Context, token string) (InviteLink, error)
	RevokeInvite(ctx context.Context, id string, at time.Time) error
	UpdateRoomMeta(ctx context.Context, conversationID, title, description, visibility string) error
	UpsertE2EDevice(ctx context.Context, dev E2EDevice) error
	RevokeE2EDevice(ctx context.Context, userID, deviceID string, at time.Time) error
	GetE2EDevice(ctx context.Context, userID, deviceID string) (E2EDevice, error)
	ListE2EDevices(ctx context.Context, userID string) ([]E2EDevice, error)
	ReplacePreKeys(ctx context.Context, userID, deviceID string, keys []E2EPreKey) error
	ConsumePreKey(ctx context.Context, userID, deviceID string) (E2EPreKey, error)
	InsertCall(ctx context.Context, call CallSession) error
	UpdateCallState(ctx context.Context, id, state string, endedAt *time.Time) error
	GetCall(ctx context.Context, id string) (CallSession, error)
	ListCalls(ctx context.Context, userID string, limit int) ([]CallSession, error)
	DeleteCall(ctx context.Context, id string) error
	DeleteCallsForUser(ctx context.Context, userID string) error
	InsertBot(ctx context.Context, bot Bot) error
	GetBotByTokenHash(ctx context.Context, hash string) (Bot, error)
	GetBotByID(ctx context.Context, id string) (Bot, error)
	GetBotByUserID(ctx context.Context, userID string) (Bot, error)
	ListBots(ctx context.Context, ownerID string) ([]Bot, error)
	DeleteBot(ctx context.Context, id string) error
	SetMemberRestriction(ctx context.Context, conversationID, userID string, canSend bool) error
	MemberCanSend(ctx context.Context, conversationID, userID string) (bool, error)
	UpsertBotGrant(ctx context.Context, grant BotGrant) error
	GetBotGrant(ctx context.Context, botID, conversationID string) (BotGrant, error)
	TouchPresence(ctx context.Context, userID string, at time.Time) error
	GetPresence(ctx context.Context, userID string) (time.Time, error)
}

type Publisher interface {
	Publish(userIDs []string, event Event)
}
