package notify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"
)

type Service struct {
	store  Store
	sender Sender
	now    func() time.Time
}

func NewService(store Store, sender Sender) *Service {
	return &Service{store: store, sender: sender, now: time.Now}
}

func (s *Service) RegisterDevice(ctx context.Context, userID, platform, token string) (Device, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	token = strings.TrimSpace(token)
	if userID == "" || token == "" || utf8.RuneCountInString(token) > MaxTokenRunes {
		return Device{}, ErrInvalidInput
	}
	switch platform {
	case "android", "ios", "web", "dev":
	default:
		return Device{}, ErrInvalidInput
	}
	existing, err := s.store.ListDevices(ctx, userID)
	if err != nil {
		return Device{}, err
	}
	for _, item := range existing {
		if item.Token == token {
			item.Platform = platform
			if err := s.store.UpsertDevice(ctx, item); err != nil {
				return Device{}, err
			}
			return item, nil
		}
	}
	if len(existing) >= MaxDevicesPerUser {
		return Device{}, ErrInvalidInput
	}
	device := Device{
		ID:        newID("dev"),
		UserID:    userID,
		Platform:  platform,
		Token:     token,
		CreatedAt: s.now().UTC(),
	}
	if err := s.store.UpsertDevice(ctx, device); err != nil {
		return Device{}, err
	}
	return device, nil
}

func (s *Service) UnregisterDevice(ctx context.Context, userID, deviceID string) error {
	if userID == "" || deviceID == "" {
		return ErrInvalidInput
	}
	return s.store.DeleteDevice(ctx, userID, deviceID)
}

func (s *Service) Preferences(ctx context.Context, userID string) (Preferences, error) {
	return s.store.GetPreferences(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, messages, calls, preview bool) (Preferences, error) {
	prefs := Preferences{
		UserID:          userID,
		MessagesEnabled: messages,
		CallsEnabled:    calls,
		ShowPreview:     preview,
		UpdatedAt:       s.now().UTC(),
	}
	if err := s.store.UpsertPreferences(ctx, prefs); err != nil {
		return Preferences{}, err
	}
	return prefs, nil
}

func (s *Service) NotifyNewMessage(ctx context.Context, senderID, senderName, conversationID, messageID, preview string, recipientIDs []string) {
	for _, recipientID := range recipientIDs {
		if recipientID == senderID {
			continue
		}
		prefs, err := s.store.GetPreferences(ctx, recipientID)
		if err != nil || !prefs.MessagesEnabled {
			continue
		}
		body := "New message"
		if prefs.ShowPreview {
			body = preview
			if utf8.RuneCountInString(body) > 140 {
				runes := []rune(body)
				body = string(runes[:140])
			}
		}
		payload := Payload{
			Title:          senderName,
			Body:           body,
			ConversationID: conversationID,
			MessageID:      messageID,
		}
		devices, err := s.store.ListDevices(ctx, recipientID)
		if err != nil || s.sender == nil {
			continue
		}
		for _, device := range devices {
			_ = s.sender.Send(ctx, device, payload)
		}
	}
}

func newID(prefix string) string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return prefix + "_" + hex.EncodeToString(buf)
}
