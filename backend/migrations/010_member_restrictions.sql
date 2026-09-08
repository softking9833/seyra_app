CREATE TABLE IF NOT EXISTS member_restrictions (
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    can_send BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (conversation_id, user_id)
);
