package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) CreateUser(ctx context.Context, user User) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, username, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`, user.ID, user.Username, user.PasswordHash, user.CreatedAt)
	if isUniqueViolation(err) {
		return ErrUsernameTaken
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, created_at, COALESCE(display_name, ''), COALESCE(bio, ''), COALESCE(avatar_key, '')
		FROM users
		WHERE LOWER(username) = LOWER($1)
	`, username)
	return scanUser(row)
}

func (s *PostgresStore) SearchUsers(ctx context.Context, query, excludeUserID string, limit int) ([]User, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, username, created_at
		FROM users
		WHERE LOWER(username) LIKE LOWER($1) || '%'
		  AND id <> $2
		ORDER BY LOWER(username)
		LIMIT $3
	`, query, excludeUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan search user: %w", err)
		}
		out = append(out, user)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, created_at, COALESCE(display_name, ''), COALESCE(bio, ''), COALESCE(avatar_key, '')
		FROM users
		WHERE id = $1
	`, id)
	return scanUser(row)
}

func (s *PostgresStore) CreateSession(ctx context.Context, session Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (
			id, user_id, access_token_hash, refresh_token_hash,
			expires_at, refresh_expires_at, revoked_at, created_at, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, session.ID, session.UserID, session.AccessTokenHash, session.RefreshTokenHash,
		session.ExpiresAt, session.RefreshExpiresAt, session.RevokedAt, session.CreatedAt, session.UserAgent)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetSessionByAccessHash(ctx context.Context, hash string) (Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, access_token_hash, refresh_token_hash,
		       expires_at, refresh_expires_at, revoked_at, created_at, COALESCE(user_agent, '')
		FROM sessions
		WHERE access_token_hash = $1
	`, hash)
	return scanSession(row)
}

func (s *PostgresStore) GetSessionByRefreshHash(ctx context.Context, hash string) (Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, access_token_hash, refresh_token_hash,
		       expires_at, refresh_expires_at, revoked_at, created_at, COALESCE(user_agent, '')
		FROM sessions
		WHERE refresh_token_hash = $1
	`, hash)
	return scanSession(row)
}

func (s *PostgresStore) UpdateSessionTokens(ctx context.Context, session Session) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET access_token_hash = $2,
		    refresh_token_hash = $3,
		    expires_at = $4,
		    refresh_expires_at = $5
		WHERE id = $1 AND revoked_at IS NULL
	`, session.ID, session.AccessTokenHash, session.RefreshTokenHash, session.ExpiresAt, session.RefreshExpiresAt)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) RevokeSession(ctx context.Context, sessionID string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL
	`, sessionID, at)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteUserAndSessions(ctx context.Context, userID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete user: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete sessions: %w", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete user: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.DisplayName, &user.Bio, &user.AvatarKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	return user, nil
}

func scanSession(row rowScanner) (Session, error) {
	var session Session
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.AccessTokenHash,
		&session.RefreshTokenHash,
		&session.ExpiresAt,
		&session.RefreshExpiresAt,
		&session.RevokedAt,
		&session.CreatedAt,
		&session.UserAgent,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("scan session: %w", err)
	}
	return session, nil
}

func (s *PostgresStore) ListSessions(ctx context.Context, userID string) ([]Session, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, access_token_hash, refresh_token_hash,
		       expires_at, refresh_expires_at, revoked_at, created_at, COALESCE(user_agent, '')
		FROM sessions WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

func (s *PostgresStore) SetSessionUserAgent(ctx context.Context, sessionID, userAgent string) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET user_agent = $2 WHERE id = $1`, sessionID, userAgent)
	return err
}

func (s *PostgresStore) UpdateUserProfile(ctx context.Context, userID, displayName, bio string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET display_name = $2, bio = $3 WHERE id = $1`, userID, displayName, bio)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) UpdateUsername(ctx context.Context, userID, username string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET username = $2 WHERE id = $1`, userID, username)
	if isUniqueViolation(err) {
		return ErrUsernameTaken
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SetAvatarKey(ctx context.Context, userID, avatarKey string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET avatar_key = $2 WHERE id = $1`, userID, avatarKey)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
