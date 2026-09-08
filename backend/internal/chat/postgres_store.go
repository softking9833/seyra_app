package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) CreateConversation(ctx context.Context, conv Conversation, memberIDs [2]string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin conversation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	kind := conv.Kind
	if kind == "" {
		kind = KindDirect
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO conversations (id, pair_key, kind, title, visibility, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (pair_key) WHERE pair_key IS NOT NULL DO NOTHING
	`, conv.ID, conv.PairKey, kind, conv.Title, visibilityOrPrivate(conv.Visibility), conv.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert conversation: %w", err)
	}
	conversationID := conv.ID
	if tag.RowsAffected() == 0 {
		var existing string
		if err := tx.QueryRow(ctx, `SELECT id FROM conversations WHERE pair_key = $1`, conv.PairKey).Scan(&existing); err != nil {
			return fmt.Errorf("load existing conversation: %w", err)
		}
		conversationID = existing
	}
	for _, userID := range memberIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO conversation_members (conversation_id, user_id, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (conversation_id, user_id) DO NOTHING
		`, conversationID, userID, conv.CreatedAt); err != nil {
			return fmt.Errorf("insert member: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit conversation: %w", err)
	}
	return nil
}

func (s *PostgresStore) CreateRoom(ctx context.Context, conv Conversation, ownerID string, memberIDs []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin room: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO conversations (id, pair_key, kind, title, visibility, created_at)
		VALUES ($1, NULL, $2, $3, $4, $5)
	`, conv.ID, conv.Kind, conv.Title, visibilityOrPrivate(conv.Visibility), conv.CreatedAt); err != nil {
		return fmt.Errorf("insert room: %w", err)
	}
	for _, userID := range memberIDs {
		role := RoleMember
		if userID == ownerID {
			role = RoleOwner
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO conversation_members (conversation_id, user_id, created_at, role)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (conversation_id, user_id) DO NOTHING
		`, conv.ID, userID, conv.CreatedAt, role); err != nil {
			return fmt.Errorf("insert room member: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit room: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetConversationByPairKey(ctx context.Context, pairKey string) (Conversation, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, pair_key, kind, title, visibility, created_at FROM conversations WHERE pair_key = $1
	`, pairKey)
	return scanConversation(row)
}

func (s *PostgresStore) GetConversation(ctx context.Context, id string) (Conversation, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, pair_key, kind, title, visibility, created_at FROM conversations WHERE id = $1
	`, id)
	return scanConversation(row)
}

func (s *PostgresStore) IsMember(ctx context.Context, conversationID, userID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM conversation_members
			WHERE conversation_id = $1 AND user_id = $2
		)
	`, conversationID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is member: %w", err)
	}
	return exists, nil
}

func (s *PostgresStore) MemberRole(ctx context.Context, conversationID, userID string) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT role FROM conversation_members
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrForbidden
	}
	if err != nil {
		return "", fmt.Errorf("member role: %w", err)
	}
	return role, nil
}

func (s *PostgresStore) ListMembers(ctx context.Context, conversationID string) ([]Member, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT conversation_id, user_id, role, last_read_at, created_at
		FROM conversation_members
		WHERE conversation_id = $1
		ORDER BY
			CASE role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END,
			created_at ASC
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ConversationID, &m.UserID, &m.Role, &m.LastReadAt, &m.CreatedAt); err != nil {
			return nil, err
		}
		if m.Role == "" {
			m.Role = RoleMember
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *PostgresStore) AddMember(ctx context.Context, conversationID, userID, role string, at time.Time) error {
	if role == "" {
		role = RoleMember
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO conversation_members (conversation_id, user_id, created_at, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`, conversationID, userID, at, role)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyMember
	}
	return nil
}

func (s *PostgresStore) RemoveMember(ctx context.Context, conversationID, userID string) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM conversation_members
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SetMemberRole(ctx context.Context, conversationID, userID, role string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE conversation_members SET role = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID, role)
	if err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) MemberIDs(ctx context.Context, conversationID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id FROM conversation_members WHERE conversation_id = $1
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("member ids: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *PostgresStore) ListSummaries(ctx context.Context, userID string) ([]ConversationSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			c.id,
			c.kind,
			c.title,
			c.visibility,
			c.created_at,
			COALESCE(peer.id, ''),
			COALESCE(peer.username, ''),
			COALESCE(last.body, ''),
			COALESCE(last.created_at, c.created_at),
			(
				SELECT COUNT(*)::int
				FROM messages m
				WHERE m.conversation_id = c.id
				  AND m.deleted_at IS NULL
				  AND m.sender_id <> $1
				  AND (me.last_read_at IS NULL OR m.created_at > me.last_read_at)
			) AS unread,
			(
				SELECT COUNT(*)::int
				FROM conversation_members cm
				WHERE cm.conversation_id = c.id
			) AS member_count
		FROM conversation_members me
		JOIN conversations c ON c.id = me.conversation_id
		LEFT JOIN LATERAL (
			SELECT u.id, u.username
			FROM conversation_members other
			JOIN users u ON u.id = other.user_id
			WHERE other.conversation_id = c.id
			  AND other.user_id <> $1
			  AND c.kind = 'direct'
			LIMIT 1
		) peer ON true
		LEFT JOIN LATERAL (
			SELECT CASE WHEN COALESCE(e2e, false) THEN 'Encrypted message' ELSE body END AS body, created_at
			FROM messages
			WHERE conversation_id = c.id AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		) last ON true
		WHERE me.user_id = $1
		ORDER BY COALESCE(last.created_at, c.created_at) DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()
	var out []ConversationSummary
	for rows.Next() {
		var item ConversationSummary
		if err := rows.Scan(
			&item.ID,
			&item.Kind,
			&item.Title,
			&item.Visibility,
			&item.CreatedAt,
			&item.Peer.ID,
			&item.Peer.Username,
			&item.LastMessagePreview,
			&item.LastMessageAt,
			&item.UnreadCount,
			&item.MemberCount,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *PostgresStore) InsertMessage(ctx context.Context, message Message) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO messages (id, conversation_id, sender_id, body, created_at, reply_to_id, attachment_id, e2e, forwarded_from_id)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, NULLIF($9, ''))
	`, message.ID, message.ConversationID, message.SenderID, message.Body, message.CreatedAt, message.ReplyToID, message.AttachmentID, message.E2E, message.ForwardedFromID)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListMessages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error) {
	var rows pgx.Rows
	var err error
	if beforeID == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, conversation_id, COALESCE(sender_id, ''), body, created_at, deleted_at, COALESCE(reply_to_id, ''), COALESCE(attachment_id, ''), edited_at, COALESCE(e2e, false), COALESCE(forwarded_from_id, '')
			FROM messages
			WHERE conversation_id = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $2
		`, conversationID, limit)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, conversation_id, COALESCE(sender_id, ''), body, created_at, deleted_at, COALESCE(reply_to_id, ''), COALESCE(attachment_id, ''), edited_at, COALESCE(e2e, false), COALESCE(forwarded_from_id, '')
			FROM messages
			WHERE conversation_id = $1
			  AND deleted_at IS NULL
			  AND created_at < (SELECT created_at FROM messages WHERE id = $2 AND conversation_id = $1)
			ORDER BY created_at DESC
			LIMIT $3
		`, conversationID, beforeID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	var newestFirst []Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		newestFirst = append(newestFirst, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]Message, 0, len(newestFirst))
	for i := len(newestFirst) - 1; i >= 0; i-- {
		out = append(out, newestFirst[i])
	}
	return out, nil
}

func (s *PostgresStore) GetMessage(ctx context.Context, id string) (Message, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, conversation_id, COALESCE(sender_id, ''), body, created_at, deleted_at, COALESCE(reply_to_id, ''), COALESCE(attachment_id, ''), edited_at, COALESCE(e2e, false), COALESCE(forwarded_from_id, '')
		FROM messages WHERE id = $1
	`, id)
	return scanMessage(row)
}

func (s *PostgresStore) SoftDeleteMessage(ctx context.Context, id string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE messages SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL
	`, id, at)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) MarkRead(ctx context.Context, conversationID, userID string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE conversation_members
		SET last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID, at)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanConversation(row scanner) (Conversation, error) {
	var conv Conversation
	var pairKey sql.NullString
	err := row.Scan(&conv.ID, &pairKey, &conv.Kind, &conv.Title, &conv.Visibility, &conv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("scan conversation: %w", err)
	}
	conv.PairKey = pairKey.String
	if conv.Kind == "" {
		conv.Kind = KindDirect
	}
	if conv.Visibility == "" {
		conv.Visibility = VisibilityPrivate
	}
	return conv, nil
}

func visibilityOrPrivate(value string) string {
	if value == "" {
		return VisibilityPrivate
	}
	return value
}

func scanMessage(row scanner) (Message, error) {
	var msg Message
	err := row.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Body, &msg.CreatedAt, &msg.DeletedAt, &msg.ReplyToID, &msg.AttachmentID, &msg.EditedAt, &msg.E2E, &msg.ForwardedFromID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, fmt.Errorf("scan message: %w", err)
	}
	return msg, nil
}

func (s *PostgresStore) UpdateMessageBody(ctx context.Context, id, body string, editedAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE messages SET body = $2, edited_at = $3 WHERE id = $1 AND deleted_at IS NULL
	`, id, body, editedAt)
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) InsertAttachment(ctx context.Context, att Attachment) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO attachments (id, conversation_id, uploader_id, object_key, filename, content_type, byte_size, created_at, e2e)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, att.ID, att.ConversationID, att.UploaderID, att.ObjectKey, att.Filename, att.ContentType, att.ByteSize, att.CreatedAt, att.E2E)
	if err != nil {
		return fmt.Errorf("insert attachment: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetAttachment(ctx context.Context, id string) (Attachment, error) {
	var att Attachment
	err := s.pool.QueryRow(ctx, `
		SELECT id, conversation_id, uploader_id, object_key, filename, content_type, byte_size, created_at, COALESCE(e2e, false)
		FROM attachments WHERE id = $1
	`, id).Scan(&att.ID, &att.ConversationID, &att.UploaderID, &att.ObjectKey, &att.Filename, &att.ContentType, &att.ByteSize, &att.CreatedAt, &att.E2E)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	if err != nil {
		return Attachment{}, fmt.Errorf("get attachment: %w", err)
	}
	return att, nil
}

func (s *PostgresStore) ListAttachments(ctx context.Context, conversationID string, limit int) ([]Attachment, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, conversation_id, uploader_id, object_key, filename, content_type, byte_size, created_at, COALESCE(e2e, false)
		FROM attachments WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()
	var out []Attachment
	for rows.Next() {
		var att Attachment
		if err := rows.Scan(&att.ID, &att.ConversationID, &att.UploaderID, &att.ObjectKey, &att.Filename, &att.ContentType, &att.ByteSize, &att.CreatedAt, &att.E2E); err != nil {
			return nil, err
		}
		out = append(out, att)
	}
	return out, rows.Err()
}

func (s *PostgresStore) DeleteAttachment(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SearchMessages(ctx context.Context, userID, query string, limit int) ([]Message, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.conversation_id, COALESCE(m.sender_id, ''), m.body, m.created_at, m.deleted_at,
		       COALESCE(m.reply_to_id, ''), COALESCE(m.attachment_id, ''), m.edited_at, COALESCE(m.e2e, false), COALESCE(m.forwarded_from_id, '')
		FROM messages m
		JOIN conversation_members cm ON cm.conversation_id = m.conversation_id AND cm.user_id = $1
		WHERE m.deleted_at IS NULL AND COALESCE(m.e2e, false) = FALSE AND lower(m.body) LIKE $2
		ORDER BY m.created_at DESC
		LIMIT $3
	`, userID, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

func (s *PostgresStore) SearchConversations(ctx context.Context, userID, query string, limit int) ([]ConversationSummary, error) {
	items, err := s.ListSummaries(ctx, userID)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(query)
	var out []ConversationSummary
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Title), needle) || strings.Contains(strings.ToLower(item.Peer.Username), needle) {
			out = append(out, item)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (s *PostgresStore) DiscoverChannels(ctx context.Context, query string, limit int) ([]ConversationSummary, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT id, kind, title, visibility, created_at,
		       (SELECT COUNT(*)::int FROM conversation_members cm WHERE cm.conversation_id = c.id)
		FROM conversations c
		WHERE kind = 'channel' AND visibility = 'public'
		  AND lower(title) LIKE $1
		ORDER BY created_at DESC
		LIMIT $2
	`, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("discover channels: %w", err)
	}
	defer rows.Close()
	var out []ConversationSummary
	for rows.Next() {
		var item ConversationSummary
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.Visibility, &item.CreatedAt, &item.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *PostgresStore) SetVisibility(ctx context.Context, conversationID, visibility string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE conversations SET visibility = $2 WHERE id = $1`, conversationID, visibility)
	if err != nil {
		return fmt.Errorf("set visibility: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ToggleReaction(ctx context.Context, messageID, userID, emoji string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3
	`, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("toggle reaction delete: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO message_reactions (message_id, user_id, emoji, created_at)
		VALUES ($1, $2, $3, $4)
	`, messageID, userID, emoji, at)
	if err != nil {
		return fmt.Errorf("toggle reaction insert: %w", err)
	}
	return nil
}
