-- Phase 5 media/privacy columns. Additive and compatible with existing rows.
ALTER TABLE attachments
    ADD COLUMN IF NOT EXISTS e2e BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE conversation_drafts
    ADD COLUMN IF NOT EXISTS reply_to_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS user_presence (
    user_id TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    last_seen_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS attachments_conversation_created_idx
    ON attachments (conversation_id, created_at DESC);
