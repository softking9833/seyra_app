-- Privacy, E2E public material, bots, pins, invites, mute/archive/drafts, forwards.

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS photo_attachment_id TEXT;

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS e2e BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS forwarded_from_id TEXT,
    ADD COLUMN IF NOT EXISTS pinned_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS user_privacy (
    user_id TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    last_seen_visible BOOLEAN NOT NULL DEFAULT TRUE,
    read_receipts BOOLEAN NOT NULL DEFAULT TRUE,
    typing_visible BOOLEAN NOT NULL DEFAULT TRUE,
    profile_visible BOOLEAN NOT NULL DEFAULT TRUE,
    notification_preview BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS user_blocks (
    blocker_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    blocked_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE TABLE IF NOT EXISTS user_reports (
    id TEXT PRIMARY KEY,
    reporter_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversation_pins (
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    message_id TEXT NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    pinned_by TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    pinned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (conversation_id, message_id)
);

CREATE TABLE IF NOT EXISTS conversation_mutes (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    muted BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (user_id, conversation_id)
);

CREATE TABLE IF NOT EXISTS conversation_archives (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, conversation_id)
);

CREATE TABLE IF NOT EXISTS conversation_drafts (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, conversation_id)
);

CREATE TABLE IF NOT EXISTS invite_links (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    created_by TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS invite_links_conv_idx ON invite_links (conversation_id);

CREATE TABLE IF NOT EXISTS e2e_devices (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    registration_id INTEGER NOT NULL,
    identity_public TEXT NOT NULL,
    signed_prekey_id INTEGER NOT NULL,
    signed_prekey_public TEXT NOT NULL,
    signed_prekey_sig TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, device_id)
);

CREATE TABLE IF NOT EXISTS e2e_prekeys (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    key_id INTEGER NOT NULL,
    public_key TEXT NOT NULL,
    PRIMARY KEY (user_id, device_id, key_id)
);

CREATE TABLE IF NOT EXISTS bots (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    username TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bot_grants (
    bot_id TEXT NOT NULL REFERENCES bots (id) ON DELETE CASCADE,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    can_read BOOLEAN NOT NULL DEFAULT TRUE,
    can_send BOOLEAN NOT NULL DEFAULT TRUE,
    can_manage_messages BOOLEAN NOT NULL DEFAULT FALSE,
    can_manage_members BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (bot_id, conversation_id)
);

CREATE INDEX IF NOT EXISTS call_sessions_conversation_idx ON call_sessions (conversation_id, created_at DESC);
