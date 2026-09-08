CREATE TABLE IF NOT EXISTS message_reactions (
    message_id TEXT NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    emoji TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji),
    CONSTRAINT message_reactions_emoji_check CHECK (char_length(emoji) BETWEEN 1 AND 16)
);

CREATE INDEX IF NOT EXISTS message_reactions_message_idx ON message_reactions (message_id);

CREATE TABLE IF NOT EXISTS call_sessions (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    caller_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('voice', 'video')),
    state TEXT NOT NULL CHECK (state IN ('ringing', 'active', 'ended', 'missed', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS call_participants (
    call_id TEXT NOT NULL REFERENCES call_sessions (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (call_id, user_id)
);

CREATE INDEX IF NOT EXISTS call_sessions_caller_idx ON call_sessions (caller_id, created_at DESC);
CREATE INDEX IF NOT EXISTS call_participants_user_idx ON call_participants (user_id, call_id);
