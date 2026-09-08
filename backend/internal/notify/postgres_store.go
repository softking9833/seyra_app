package notify

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) UpsertDevice(ctx context.Context, device Device) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification_devices (id, user_id, platform, token, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, token) DO UPDATE SET platform = EXCLUDED.platform
	`, device.ID, device.UserID, device.Platform, device.Token, device.CreatedAt)
	if err != nil {
		return fmt.Errorf("upsert device: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteDevice(ctx context.Context, userID, deviceID string) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM notification_devices WHERE id = $1 AND user_id = $2
	`, deviceID, userID)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListDevices(ctx context.Context, userID string) ([]Device, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, platform, token, created_at
		FROM notification_devices
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, MaxDevicesPerUser)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		var item Device
		if err := rows.Scan(&item.ID, &item.UserID, &item.Platform, &item.Token, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT user_id, messages_enabled, calls_enabled, show_preview, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`, userID)
	var prefs Preferences
	err := row.Scan(&prefs.UserID, &prefs.MessagesEnabled, &prefs.CallsEnabled, &prefs.ShowPreview, &prefs.UpdatedAt)
	if errorsIsNoRows(err) {
		return Preferences{
			UserID:          userID,
			MessagesEnabled: true,
			CallsEnabled:    true,
			ShowPreview:     true,
		}, nil
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("get preferences: %w", err)
	}
	return prefs, nil
}

func (s *PostgresStore) UpsertPreferences(ctx context.Context, prefs Preferences) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification_preferences (user_id, messages_enabled, calls_enabled, show_preview, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			messages_enabled = EXCLUDED.messages_enabled,
			calls_enabled = EXCLUDED.calls_enabled,
			show_preview = EXCLUDED.show_preview,
			updated_at = EXCLUDED.updated_at
	`, prefs.UserID, prefs.MessagesEnabled, prefs.CallsEnabled, prefs.ShowPreview, prefs.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert preferences: %w", err)
	}
	return nil
}

func errorsIsNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
