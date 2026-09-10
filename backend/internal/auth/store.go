package auth

import (
	"context"
	"time"
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	DisplayName  string
	Bio          string
	AvatarKey    string
	CreatedAt    time.Time
}

type Session struct {
	ID               string
	UserID           string
	AccessTokenHash  string
	RefreshTokenHash string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UserAgent        string
}

type Store interface {
	CreateUser(ctx context.Context, user User) error
	GetUserByUsername(ctx context.Context, username string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	SearchUsers(ctx context.Context, query, excludeUserID string, limit int) ([]User, error)
	CreateSession(ctx context.Context, session Session) error
	GetSessionByAccessHash(ctx context.Context, hash string) (Session, error)
	GetSessionByRefreshHash(ctx context.Context, hash string) (Session, error)
	UpdateSessionTokens(ctx context.Context, session Session) error
	RevokeSession(ctx context.Context, sessionID string, at time.Time) error
	ListSessions(ctx context.Context, userID string) ([]Session, error)
	SetSessionUserAgent(ctx context.Context, sessionID, userAgent string) error
	UpdateUserProfile(ctx context.Context, userID, displayName, bio string) error
	UpdateUsername(ctx context.Context, userID, username string) error
	SetAvatarKey(ctx context.Context, userID, avatarKey string) error
	DeleteUserAndSessions(ctx context.Context, userID string) error
}
