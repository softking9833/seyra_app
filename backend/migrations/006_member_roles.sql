-- Membership roles for groups and channels. Direct chats keep default 'member'.
ALTER TABLE conversation_members
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'member';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'conversation_members_role_check'
    ) THEN
        ALTER TABLE conversation_members
            ADD CONSTRAINT conversation_members_role_check
            CHECK (role IN ('owner', 'admin', 'member'));
    END IF;
END $$;

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'private';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'conversations_visibility_check'
    ) THEN
        ALTER TABLE conversations
            ADD CONSTRAINT conversations_visibility_check
            CHECK (visibility IN ('private', 'public'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS conversations_kind_visibility_idx
    ON conversations (kind, visibility)
    WHERE kind = 'channel' AND visibility = 'public';
