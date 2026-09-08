-- Device tokens for push dispatch. Tokens are secrets: never log them.
CREATE TABLE IF NOT EXISTS notification_devices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    token TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, token)
);

CREATE INDEX IF NOT EXISTS notification_devices_user_id_idx
    ON notification_devices (user_id);

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    messages_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    calls_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    show_preview BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
