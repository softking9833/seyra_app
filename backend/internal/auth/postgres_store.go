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
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE LOWER(username) = LOWER($1)
	`, username)
	return scanUser(row)
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE id = $1
	`, id)
	return scanUser(row)
}

func (s *PostgresStore) CreateSession(ctx context.Context, session Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (
			id, user_id, access_token_hash, refresh_token_hash,
			expires_at, refresh_expires_at, revoked_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, session.ID, session.UserID, session.AccessTokenHash, session.RefreshTokenHash,
		session.ExpiresAt, session.RefreshExpiresAt, session.RevokedAt, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetSessionByAccessHash(ctx context.Context, hash string) (Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, access_token_hash, refresh_token_hash,
		       expires_at, refresh_expires_at, revoked_at, created_at
		FROM sessions
		WHERE access_token_hash = $1
	`, hash)
	return scanSession(row)
}

func (s *PostgresStore) GetSessionByRefreshHash(ctx context.Context, hash string) (Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, access_token_hash, refresh_token_hash,
		       expires_at, refresh_expires_at, revoked_at, created_at
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

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
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
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("scan session: %w", err)
	}
	return session, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
