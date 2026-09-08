package notify

import (
	"context"
	"testing"
)

func TestRegisterDeviceAndNotifyRespectsPreferences(t *testing.T) {
	store := NewMemoryStore()
	sender := &RecordingSender{}
	svc := NewService(store, sender)
	ctx := context.Background()

	if _, err := svc.RegisterDevice(ctx, "usr_lin", "android", "token-lin"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RegisterDevice(ctx, "usr_lin", "unknown", "x"); err != ErrInvalidInput {
		t.Fatalf("expected invalid platform, got %v", err)
	}

	svc.NotifyNewMessage(ctx, "usr_ada", "ada", "cht_1", "msg_1", "secret text", []string{"usr_lin"})
	if sender.Count != 1 {
		t.Fatalf("expected one push, got %d", sender.Count)
	}

	if _, err := svc.UpdatePreferences(ctx, "usr_lin", false, true, true); err != nil {
		t.Fatal(err)
	}
	svc.NotifyNewMessage(ctx, "usr_ada", "ada", "cht_1", "msg_2", "muted", []string{"usr_lin"})
	if sender.Count != 1 {
		t.Fatalf("muted user still notified: %d", sender.Count)
	}
}

func TestWebhookSenderSkipsWhenUnconfigured(t *testing.T) {
	sender := &WebhookSender{}
	if err := sender.Send(context.Background(), Device{Token: "secret"}, Payload{Title: "ada"}); err != nil {
		t.Fatal(err)
	}
}
