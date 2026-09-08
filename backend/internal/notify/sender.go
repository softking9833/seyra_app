package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NoopSender is used when no push gateway is configured. Device registration
// still works; OS push is skipped until SEYRA_PUSH_WEBHOOK_URL is set.
type NoopSender struct{}

func (NoopSender) Send(_ context.Context, _ Device, _ Payload) error {
	return nil
}

// RecordingSender captures dispatches for tests. It never logs tokens.
type RecordingSender struct {
	Count int
}

func (s *RecordingSender) Send(_ context.Context, _ Device, _ Payload) error {
	s.Count++
	return nil
}

// WebhookSender posts to an operator-configured gateway (FCM/APNs adapter).
// Domain code does not import a vendor SDK. The token is sent only to that URL.
type WebhookSender struct {
	URL    string
	Secret string
	Client *http.Client
}

func (s *WebhookSender) Send(ctx context.Context, device Device, payload Payload) error {
	if s.URL == "" {
		return nil
	}
	body, err := json.Marshal(map[string]string{
		"platform":        device.Platform,
		"token":           device.Token,
		"title":           payload.Title,
		"body":            payload.Body,
		"conversation_id": payload.ConversationID,
		"message_id":      payload.MessageID,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+s.Secret)
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("push webhook status %d", res.StatusCode)
	}
	return nil
}

type MultiSender struct {
	Senders []Sender
}

func (s MultiSender) Send(ctx context.Context, device Device, payload Payload) error {
	var first error
	for _, item := range s.Senders {
		if err := item.Send(ctx, device, payload); err != nil && first == nil {
			first = err
		}
	}
	return first
}
