package notify

import "context"

type Store interface {
	UpsertDevice(ctx context.Context, device Device) error
	DeleteDevice(ctx context.Context, userID, deviceID string) error
	ListDevices(ctx context.Context, userID string) ([]Device, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, prefs Preferences) error
}
