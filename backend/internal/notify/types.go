package notify

import (
	"context"
	"time"
)

const (
	MaxTokenRunes     = 4096
	MaxDevicesPerUser = 20
)

type Device struct {
	ID        string
	UserID    string
	Platform  string
	Token     string
	CreatedAt time.Time
}

type Preferences struct {
	UserID          string
	MessagesEnabled bool
	CallsEnabled    bool
	ShowPreview     bool
	UpdatedAt       time.Time
}

type Payload struct {
	Title          string
	Body           string
	ConversationID string
	MessageID      string
}

type Sender interface {
	Send(ctx context.Context, device Device, payload Payload) error
}
