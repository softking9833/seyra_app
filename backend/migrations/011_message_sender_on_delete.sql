-- Account deletion must not fail after a user has sent messages.
-- Keep message history for remaining members; drop the sender reference.
ALTER TABLE messages
    ALTER COLUMN sender_id DROP NOT NULL;

ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_sender_id_fkey;

ALTER TABLE messages
    ADD CONSTRAINT messages_sender_id_fkey
    FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE SET NULL;
