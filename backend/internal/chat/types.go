package chat

import "time"

// UserRef is the only user data chat is allowed to keep. Never include
// password hashes, tokens, or session secrets.
type UserRef struct {
	ID       string
	Username string
}

const (
	KindDirect          = "direct"
	KindGroup           = "group"
	KindChannel         = "channel"
	MaxTitleRunes       = 80
	MaxRoomMembers      = 50
	RoleOwner           = "owner"
	RoleAdmin           = "admin"
	RoleMember          = "member"
	VisibilityPrivate   = "private"
	VisibilityPublic    = "public"
	EventMessageCreated = "message.created"
	EventMessageDeleted = "message.deleted"
	EventMemberJoined   = "member.joined"
	EventMemberLeft     = "member.left"
	EventMemberUpdated  = "member.updated"
	MaxMessageRunes     = 4000
	DefaultPageSize     = 50
	MaxPageSize         = 100
	SearchLimit         = 20
	EventMessageUpdated = "message.updated"
	EventCallSignal     = "call.signal"
	EventPinned         = "message.pinned"
	EventUnpinned       = "message.unpinned"
	EventTyping         = "typing"
)

type PrivacySettings struct {
	UserID               string
	LastSeenVisible      bool
	ReadReceipts         bool
	TypingVisible        bool
	ProfileVisible       bool
	NotificationPreview  bool
}

type InviteLink struct {
	ID             string
	ConversationID string
	Token          string
	CreatedBy      string
	CreatedAt      time.Time
	RevokedAt      *time.Time
}

type E2EDevice struct {
	UserID             string
	DeviceID           string
	RegistrationID     int
	IdentityPublic     string
	SignedPreKeyID     int
	SignedPreKeyPublic string
	SignedPreKeySig    string
	CreatedAt          time.Time
	RevokedAt          *time.Time
}

type E2EPreKey struct {
	UserID    string
	DeviceID  string
	KeyID     int
	PublicKey string
}

type CallSession struct {
	ID             string
	ConversationID string
	CallerID       string
	Kind           string
	State          string
	CreatedAt      time.Time
	EndedAt        *time.Time
	ParticipantIDs []string
}

type Bot struct {
	ID        string
	OwnerID   string
	UserID    string
	Username  string
	TokenHash string
	CreatedAt time.Time
}

type BotGrant struct {
	BotID              string
	ConversationID     string
	CanRead            bool
	CanSend            bool
	CanManageMessages  bool
	CanManageMembers   bool
}

type Conversation struct {
	ID         string
	PairKey    string
	Kind       string
	Title      string
	Visibility  string
	Description string
	CreatedAt   time.Time
}

type Member struct {
	ConversationID string
	UserID         string
	Role           string
	LastReadAt     *time.Time
	CreatedAt      time.Time
}

type RoomMember struct {
	User UserRef
	Role string
}

type Message struct {
	ID             string
	ConversationID string
	SenderID       string
	Body           string
	ReplyToID      string
	AttachmentID   string
	E2E            bool
	ForwardedFromID string
	CreatedAt      time.Time
	EditedAt       *time.Time
	DeletedAt      *time.Time
}

type Attachment struct {
	ID             string
	ConversationID string
	UploaderID     string
	ObjectKey      string
	Filename       string
	ContentType    string
	ByteSize       int64
	E2E            bool
	CreatedAt      time.Time
}

type SearchHit struct {
	Kind    string
	ID      string
	Title   string
	Snippet string
}

type ConversationSummary struct {
	ID                 string
	Kind               string
	Title              string
	Visibility         string
	MemberCount        int
	Peer               UserRef
	LastMessagePreview string
	LastMessageAt      time.Time
	UnreadCount        int
	CreatedAt          time.Time
}

type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}
