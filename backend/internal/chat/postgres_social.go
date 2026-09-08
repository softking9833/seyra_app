package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) IsBlocked(ctx context.Context, actorID, otherID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1)
		)
	`, actorID, otherID).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) BlockUser(ctx context.Context, actorID, otherID string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_blocks (blocker_id, blocked_id, created_at) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, actorID, otherID, at)
	return err
}

func (s *PostgresStore) UnblockUser(ctx context.Context, actorID, otherID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2`, actorID, otherID)
	return err
}

func (s *PostgresStore) ListBlocked(ctx context.Context, actorID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT blocked_id FROM user_blocks WHERE blocker_id = $1`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ReportUser(ctx context.Context, id, reporterID, targetID, reason string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_reports (id, reporter_id, target_id, reason, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, reporterID, targetID, reason, at)
	return err
}

func (s *PostgresStore) GetPrivacy(ctx context.Context, userID string) (PrivacySettings, error) {
	var p PrivacySettings
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, last_seen_visible, read_receipts, typing_visible, profile_visible, notification_preview
		FROM user_privacy WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.LastSeenVisible, &p.ReadReceipts, &p.TypingVisible, &p.ProfileVisible, &p.NotificationPreview)
	if errors.Is(err, pgx.ErrNoRows) {
		return PrivacySettings{
			UserID: userID, LastSeenVisible: true, ReadReceipts: true,
			TypingVisible: true, ProfileVisible: true, NotificationPreview: true,
		}, nil
	}
	return p, err
}

func (s *PostgresStore) PutPrivacy(ctx context.Context, settings PrivacySettings) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_privacy (user_id, last_seen_visible, read_receipts, typing_visible, profile_visible, notification_preview)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			last_seen_visible = EXCLUDED.last_seen_visible,
			read_receipts = EXCLUDED.read_receipts,
			typing_visible = EXCLUDED.typing_visible,
			profile_visible = EXCLUDED.profile_visible,
			notification_preview = EXCLUDED.notification_preview
	`, settings.UserID, settings.LastSeenVisible, settings.ReadReceipts, settings.TypingVisible, settings.ProfileVisible, settings.NotificationPreview)
	return err
}

func (s *PostgresStore) PinMessage(ctx context.Context, conversationID, messageID, actorID string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO conversation_pins (conversation_id, message_id, pinned_by, pinned_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, conversationID, messageID, actorID, at)
	return err
}

func (s *PostgresStore) UnpinMessage(ctx context.Context, conversationID, messageID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM conversation_pins WHERE conversation_id = $1 AND message_id = $2`, conversationID, messageID)
	return err
}

func (s *PostgresStore) ListPins(ctx context.Context, conversationID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT message_id FROM conversation_pins WHERE conversation_id = $1 ORDER BY pinned_at DESC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *PostgresStore) SetMutedConv(ctx context.Context, userID, conversationID string, muted bool) error {
	if !muted {
		_, err := s.pool.Exec(ctx, `DELETE FROM conversation_mutes WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID)
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO conversation_mutes (user_id, conversation_id, muted) VALUES ($1, $2, TRUE)
		ON CONFLICT (user_id, conversation_id) DO UPDATE SET muted = TRUE
	`, userID, conversationID)
	return err
}

func (s *PostgresStore) IsMutedConv(ctx context.Context, userID, conversationID string) (bool, error) {
	var muted bool
	err := s.pool.QueryRow(ctx, `SELECT muted FROM conversation_mutes WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID).Scan(&muted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return muted, err
}

func (s *PostgresStore) SetArchived(ctx context.Context, userID, conversationID string, archived bool) error {
	if !archived {
		_, err := s.pool.Exec(ctx, `DELETE FROM conversation_archives WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID)
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO conversation_archives (user_id, conversation_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, conversationID)
	return err
}

func (s *PostgresStore) SaveDraft(ctx context.Context, userID, conversationID, body string, at time.Time) error {
	if body == "" {
		_, err := s.pool.Exec(ctx, `DELETE FROM conversation_drafts WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID)
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO conversation_drafts (user_id, conversation_id, body, updated_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, conversation_id) DO UPDATE SET body = EXCLUDED.body, updated_at = EXCLUDED.updated_at
	`, userID, conversationID, body, at)
	return err
}

func (s *PostgresStore) GetDraft(ctx context.Context, userID, conversationID string) (string, error) {
	var body string
	err := s.pool.QueryRow(ctx, `SELECT body FROM conversation_drafts WHERE user_id = $1 AND conversation_id = $2`, userID, conversationID).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return body, err
}

func (s *PostgresStore) CreateInvite(ctx context.Context, link InviteLink) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO invite_links (id, conversation_id, token, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, link.ID, link.ConversationID, link.Token, link.CreatedBy, link.CreatedAt)
	return err
}

func (s *PostgresStore) GetInviteByToken(ctx context.Context, token string) (InviteLink, error) {
	var link InviteLink
	err := s.pool.QueryRow(ctx, `
		SELECT id, conversation_id, token, created_by, created_at, revoked_at
		FROM invite_links WHERE token = $1
	`, token).Scan(&link.ID, &link.ConversationID, &link.Token, &link.CreatedBy, &link.CreatedAt, &link.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return InviteLink{}, ErrNotFound
	}
	return link, err
}

func (s *PostgresStore) RevokeInvite(ctx context.Context, id string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE invite_links SET revoked_at = $2 WHERE id = $1`, id, at)
	return err
}

func (s *PostgresStore) UpdateRoomMeta(ctx context.Context, conversationID, title, description, visibility string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE conversations SET
			title = CASE WHEN $2 = '' THEN title ELSE $2 END,
			description = $3,
			visibility = CASE WHEN $4 = '' THEN visibility ELSE $4 END
		WHERE id = $1
	`, conversationID, title, description, visibility)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) UpsertE2EDevice(ctx context.Context, dev E2EDevice) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO e2e_devices (user_id, device_id, registration_id, identity_public, signed_prekey_id, signed_prekey_public, signed_prekey_sig, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, device_id) DO UPDATE SET
			registration_id = EXCLUDED.registration_id,
			identity_public = EXCLUDED.identity_public,
			signed_prekey_id = EXCLUDED.signed_prekey_id,
			signed_prekey_public = EXCLUDED.signed_prekey_public,
			signed_prekey_sig = EXCLUDED.signed_prekey_sig,
			revoked_at = NULL
	`, dev.UserID, dev.DeviceID, dev.RegistrationID, dev.IdentityPublic, dev.SignedPreKeyID, dev.SignedPreKeyPublic, dev.SignedPreKeySig, dev.CreatedAt)
	return err
}

func (s *PostgresStore) RevokeE2EDevice(ctx context.Context, userID, deviceID string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE e2e_devices SET revoked_at = $3 WHERE user_id = $1 AND device_id = $2`, userID, deviceID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetE2EDevice(ctx context.Context, userID, deviceID string) (E2EDevice, error) {
	var d E2EDevice
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, device_id, registration_id, identity_public, signed_prekey_id, signed_prekey_public, signed_prekey_sig, created_at, revoked_at
		FROM e2e_devices WHERE user_id = $1 AND device_id = $2 AND revoked_at IS NULL
	`, userID, deviceID).Scan(&d.UserID, &d.DeviceID, &d.RegistrationID, &d.IdentityPublic, &d.SignedPreKeyID, &d.SignedPreKeyPublic, &d.SignedPreKeySig, &d.CreatedAt, &d.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return E2EDevice{}, ErrNotFound
	}
	return d, err
}

func (s *PostgresStore) ListE2EDevices(ctx context.Context, userID string) ([]E2EDevice, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, device_id, registration_id, identity_public, signed_prekey_id, signed_prekey_public, signed_prekey_sig, created_at, revoked_at
		FROM e2e_devices WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []E2EDevice
	for rows.Next() {
		var d E2EDevice
		if err := rows.Scan(&d.UserID, &d.DeviceID, &d.RegistrationID, &d.IdentityPublic, &d.SignedPreKeyID, &d.SignedPreKeyPublic, &d.SignedPreKeySig, &d.CreatedAt, &d.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ReplacePreKeys(ctx context.Context, userID, deviceID string, keys []E2EPreKey) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM e2e_prekeys WHERE user_id = $1 AND device_id = $2`, userID, deviceID); err != nil {
		return err
	}
	for _, k := range keys {
		if _, err := tx.Exec(ctx, `INSERT INTO e2e_prekeys (user_id, device_id, key_id, public_key) VALUES ($1, $2, $3, $4)`, userID, deviceID, k.KeyID, k.PublicKey); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) ConsumePreKey(ctx context.Context, userID, deviceID string) (E2EPreKey, error) {
	var k E2EPreKey
	err := s.pool.QueryRow(ctx, `
		DELETE FROM e2e_prekeys
		WHERE ctid IN (
			SELECT ctid FROM e2e_prekeys WHERE user_id = $1 AND device_id = $2 LIMIT 1
		)
		RETURNING user_id, device_id, key_id, public_key
	`, userID, deviceID).Scan(&k.UserID, &k.DeviceID, &k.KeyID, &k.PublicKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return E2EPreKey{}, ErrNotFound
	}
	return k, err
}

func (s *PostgresStore) InsertCall(ctx context.Context, call CallSession) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO call_sessions (id, conversation_id, caller_id, kind, state, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, call.ID, call.ConversationID, call.CallerID, call.Kind, call.State, call.CreatedAt); err != nil {
		return err
	}
	for _, id := range call.ParticipantIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO call_participants (call_id, user_id) VALUES ($1, $2)`, call.ID, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) UpdateCallState(ctx context.Context, id, state string, endedAt *time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE call_sessions SET state = $2, ended_at = $3 WHERE id = $1`, id, state, endedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) GetCall(ctx context.Context, id string) (CallSession, error) {
	var c CallSession
	err := s.pool.QueryRow(ctx, `
		SELECT id, conversation_id, caller_id, kind, state, created_at, ended_at
		FROM call_sessions WHERE id = $1
	`, id).Scan(&c.ID, &c.ConversationID, &c.CallerID, &c.Kind, &c.State, &c.CreatedAt, &c.EndedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CallSession{}, ErrNotFound
	}
	if err != nil {
		return CallSession{}, err
	}
	rows, err := s.pool.Query(ctx, `SELECT user_id FROM call_participants WHERE call_id = $1`, id)
	if err != nil {
		return c, nil
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			c.ParticipantIDs = append(c.ParticipantIDs, uid)
		}
	}
	return c, nil
}

func (s *PostgresStore) ListCalls(ctx context.Context, userID string, limit int) ([]CallSession, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT c.id, c.conversation_id, c.caller_id, c.kind, c.state, c.created_at, c.ended_at
		FROM call_sessions c
		LEFT JOIN call_participants p ON p.call_id = c.id
		WHERE c.caller_id = $1 OR p.user_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list calls: %w", err)
	}
	defer rows.Close()
	var out []CallSession
	for rows.Next() {
		var c CallSession
		if err := rows.Scan(&c.ID, &c.ConversationID, &c.CallerID, &c.Kind, &c.State, &c.CreatedAt, &c.EndedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PostgresStore) InsertBot(ctx context.Context, bot Bot) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO bots (id, owner_id, user_id, username, token_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, bot.ID, bot.OwnerID, bot.UserID, bot.Username, bot.TokenHash, bot.CreatedAt)
	return err
}

func (s *PostgresStore) GetBotByTokenHash(ctx context.Context, hash string) (Bot, error) {
	var b Bot
	err := s.pool.QueryRow(ctx, `
		SELECT id, owner_id, user_id, username, token_hash, created_at FROM bots WHERE token_hash = $1
	`, hash).Scan(&b.ID, &b.OwnerID, &b.UserID, &b.Username, &b.TokenHash, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bot{}, ErrNotFound
	}
	return b, err
}

func (s *PostgresStore) GetBotByID(ctx context.Context, id string) (Bot, error) {
	var b Bot
	err := s.pool.QueryRow(ctx, `
		SELECT id, owner_id, user_id, username, token_hash, created_at FROM bots WHERE id = $1
	`, id).Scan(&b.ID, &b.OwnerID, &b.UserID, &b.Username, &b.TokenHash, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bot{}, ErrNotFound
	}
	return b, err
}

func (s *PostgresStore) GetBotByUserID(ctx context.Context, userID string) (Bot, error) {
	var b Bot
	err := s.pool.QueryRow(ctx, `
		SELECT id, owner_id, user_id, username, token_hash, created_at FROM bots WHERE user_id = $1
	`, userID).Scan(&b.ID, &b.OwnerID, &b.UserID, &b.Username, &b.TokenHash, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bot{}, ErrNotFound
	}
	return b, err
}

func (s *PostgresStore) SetMemberRestriction(ctx context.Context, conversationID, userID string, canSend bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO member_restrictions (conversation_id, user_id, can_send)
		VALUES ($1, $2, $3)
		ON CONFLICT (conversation_id, user_id) DO UPDATE SET can_send = EXCLUDED.can_send
	`, conversationID, userID, canSend)
	return err
}

func (s *PostgresStore) MemberCanSend(ctx context.Context, conversationID, userID string) (bool, error) {
	var can bool
	err := s.pool.QueryRow(ctx, `
		SELECT can_send FROM member_restrictions WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID).Scan(&can)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	return can, err
}

func (s *PostgresStore) ListBots(ctx context.Context, ownerID string) ([]Bot, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, owner_id, user_id, username, token_hash, created_at FROM bots WHERE owner_id = $1`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Bot
	for rows.Next() {
		var b Bot
		if err := rows.Scan(&b.ID, &b.OwnerID, &b.UserID, &b.Username, &b.TokenHash, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *PostgresStore) DeleteBot(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM bots WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) UpsertBotGrant(ctx context.Context, grant BotGrant) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO bot_grants (bot_id, conversation_id, can_read, can_send, can_manage_messages, can_manage_members)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (bot_id, conversation_id) DO UPDATE SET
			can_read = EXCLUDED.can_read,
			can_send = EXCLUDED.can_send,
			can_manage_messages = EXCLUDED.can_manage_messages,
			can_manage_members = EXCLUDED.can_manage_members
	`, grant.BotID, grant.ConversationID, grant.CanRead, grant.CanSend, grant.CanManageMessages, grant.CanManageMembers)
	return err
}

func (s *PostgresStore) GetBotGrant(ctx context.Context, botID, conversationID string) (BotGrant, error) {
	var g BotGrant
	err := s.pool.QueryRow(ctx, `
		SELECT bot_id, conversation_id, can_read, can_send, can_manage_messages, can_manage_members
		FROM bot_grants WHERE bot_id = $1 AND conversation_id = $2
	`, botID, conversationID).Scan(&g.BotID, &g.ConversationID, &g.CanRead, &g.CanSend, &g.CanManageMessages, &g.CanManageMembers)
	if errors.Is(err, pgx.ErrNoRows) {
		return BotGrant{}, ErrNotFound
	}
	return g, err
}

func (s *PostgresStore) TouchPresence(ctx context.Context, userID string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_presence (user_id, last_seen_at) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at
	`, userID, at)
	return err
}

func (s *PostgresStore) GetPresence(ctx context.Context, userID string) (time.Time, error) {
	var at time.Time
	err := s.pool.QueryRow(ctx, `SELECT last_seen_at FROM user_presence WHERE user_id = $1`, userID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	return at, err
}
