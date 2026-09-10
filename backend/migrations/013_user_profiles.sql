-- Profile fields, session labels, photo visibility. Additive.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS avatar_key TEXT NOT NULL DEFAULT '';

ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS user_agent TEXT NOT NULL DEFAULT '';

ALTER TABLE user_privacy
    ADD COLUMN IF NOT EXISTS photo_visible BOOLEAN NOT NULL DEFAULT TRUE;
