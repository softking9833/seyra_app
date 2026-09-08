-- 1-to-1 conversations. pair_key is the sorted pair of member user IDs so the
-- same two users cannot accidentally get a second conversation.
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    pair_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversation_members (
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    last_read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS conversation_members_user_id_idx
    ON conversation_members (user_id);

-- Messages are server-readable plaintext until E2E encryption is designed.
-- Do not store auth tokens or password hashes here.
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sender_id TEXT NOT NULL REFERENCES users (id),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS messages_conversation_created_idx
    ON messages (conversation_id, created_at DESC);

CREATE INDEX IF NOT EXISTS messages_sender_id_idx
    ON messages (sender_id);
