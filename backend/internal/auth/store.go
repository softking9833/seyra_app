package auth

import (
	"context"
	"time"
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID                string
	UserID            string
	AccessTokenHash   string
	RefreshTokenHash  string
	ExpiresAt         time.Time
	RefreshExpiresAt  time.Time
	RevokedAt         *time.Time
	CreatedAt         time.Time
}

type Store interface {
	CreateUser(ctx context.Context, user User) error
	GetUserByUsername(ctx context.Context, username string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	CreateSession(ctx context.Context, session Session) error
	GetSessionByAccessHash(ctx context.Context, hash string) (Session, error)
	GetSessionByRefreshHash(ctx context.Context, hash string) (Session, error)
	UpdateSessionTokens(ctx context.Context, session Session) error
	RevokeSession(ctx context.Context, sessionID string, at time.Time) error
}
