-- Groups and channels share the conversations table. Direct chats keep a
-- unique pair_key; rooms use a title and an unbounded member set.
ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'direct',
    ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';

ALTER TABLE conversations ALTER COLUMN pair_key DROP NOT NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'conversations_pair_key_key'
    ) THEN
        ALTER TABLE conversations DROP CONSTRAINT conversations_pair_key_key;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS conversations_pair_key_uidx
    ON conversations (pair_key)
    WHERE pair_key IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'conversations_kind_check'
    ) THEN
        ALTER TABLE conversations
            ADD CONSTRAINT conversations_kind_check
            CHECK (kind IN ('direct', 'group', 'channel'));
    END IF;
END $$;
