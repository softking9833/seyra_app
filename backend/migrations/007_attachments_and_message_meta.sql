-- Attachments live in object storage; Postgres keeps metadata only.
CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    uploader_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    object_key TEXT NOT NULL UNIQUE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    byte_size BIGINT NOT NULL CHECK (byte_size > 0 AND byte_size <= 26214400),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS attachments_conversation_idx ON attachments (conversation_id);

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS reply_to_id TEXT REFERENCES messages (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS attachment_id TEXT REFERENCES attachments (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS messages_conversation_created_idx ON messages (conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS messages_conversation_body_idx ON messages (conversation_id, lower(body));

CREATE INDEX IF NOT EXISTS conversations_title_lower_idx ON conversations (lower(title));
