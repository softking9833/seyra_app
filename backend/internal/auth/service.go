package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	accessTTL  = time.Hour
	refreshTTL = 30 * 24 * time.Hour
)

var dummyPasswordHash string

func init() {
	hash, err := HashPassword("timing-guard")
	if err == nil {
		dummyPasswordHash = hash
	}
}

type Service struct {
	store  Store
	pepper string
	now    func() time.Time
}

func NewService(store Store, pepper string) *Service {
	return &Service{
		store:  store,
		pepper: pepper,
		now:    time.Now,
	}
}

type IssuedSession struct {
	User         User
	Session      Session
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

func (s *Service) Register(ctx context.Context, username, password string) (IssuedSession, error) {
	normalized, err := NormalizeUsername(username)
	if err != nil {
		return IssuedSession{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return IssuedSession{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return IssuedSession{}, fmt.Errorf("hash password: %w", err)
	}
	user := User{
		ID:           newID("usr"),
		Username:     normalized,
		PasswordHash: hash,
		CreatedAt:    s.now().UTC(),
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return IssuedSession{}, err
	}
	return s.issueSession(ctx, user)
}

func (s *Service) Login(ctx context.Context, username, password string) (IssuedSession, error) {
	normalized, err := NormalizeUsername(username)
	if err != nil {
		return IssuedSession{}, ErrInvalidCredentials
	}
	if err := ValidatePassword(password); err != nil {
		return IssuedSession{}, ErrInvalidCredentials
	}
	user, err := s.store.GetUserByUsername(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			if dummyPasswordHash != "" {
				_, _ = ComparePassword(dummyPasswordHash, password)
			}
			return IssuedSession{}, ErrInvalidCredentials
		}
		return IssuedSession{}, err
	}
	ok, err := ComparePassword(user.PasswordHash, password)
	if err != nil || !ok {
		return IssuedSession{}, ErrInvalidCredentials
	}
	return s.issueSession(ctx, user)
}

func (s *Service) CurrentSession(ctx context.Context, accessToken string) (User, Session, error) {
	session, err := s.sessionByAccess(ctx, accessToken)
	if err != nil {
		return User{}, Session{}, err
	}
	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil {
		return User{}, Session{}, err
	}
	return user, session, nil
}

func (s *Service) Logout(ctx context.Context, accessToken string) error {
	session, err := s.sessionByAccess(ctx, accessToken)
	if err != nil {
		return err
	}
	return s.store.RevokeSession(ctx, session.ID, s.now().UTC())
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (IssuedSession, error) {
	if refreshToken == "" {
		return IssuedSession{}, ErrUnauthorized
	}
	session, err := s.store.GetSessionByRefreshHash(ctx, HashToken(s.pepper, refreshToken))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return IssuedSession{}, ErrUnauthorized
		}
		return IssuedSession{}, err
	}
	now := s.now().UTC()
	if session.RevokedAt != nil {
		return IssuedSession{}, ErrUnauthorized
	}
	if now.After(session.RefreshExpiresAt) {
		return IssuedSession{}, ErrSessionExpired
	}
	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil {
		return IssuedSession{}, err
	}
	access, err := NewOpaqueToken()
	if err != nil {
		return IssuedSession{}, err
	}
	refresh, err := NewOpaqueToken()
	if err != nil {
		return IssuedSession{}, err
	}
	session.AccessTokenHash = HashToken(s.pepper, access)
	session.RefreshTokenHash = HashToken(s.pepper, refresh)
	session.ExpiresAt = now.Add(accessTTL)
	session.RefreshExpiresAt = now.Add(refreshTTL)
	if err := s.store.UpdateSessionTokens(ctx, session); err != nil {
		return IssuedSession{}, err
	}
	return IssuedSession{
		User:         user,
		Session:      session,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(accessTTL.Seconds()),
	}, nil
}

func (s *Service) issueSession(ctx context.Context, user User) (IssuedSession, error) {
	access, err := NewOpaqueToken()
	if err != nil {
		return IssuedSession{}, err
	}
	refresh, err := NewOpaqueToken()
	if err != nil {
		return IssuedSession{}, err
	}
	now := s.now().UTC()
	session := Session{
		ID:               newID("ses"),
		UserID:           user.ID,
		AccessTokenHash:  HashToken(s.pepper, access),
		RefreshTokenHash: HashToken(s.pepper, refresh),
		ExpiresAt:        now.Add(accessTTL),
		RefreshExpiresAt: now.Add(refreshTTL),
		CreatedAt:        now,
	}
	if err := s.store.CreateSession(ctx, session); err != nil {
		return IssuedSession{}, err
	}
	return IssuedSession{
		User:         user,
		Session:      session,
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(accessTTL.Seconds()),
	}, nil
}

func (s *Service) sessionByAccess(ctx context.Context, accessToken string) (Session, error) {
	if accessToken == "" {
		return Session{}, ErrUnauthorized
	}
	session, err := s.store.GetSessionByAccessHash(ctx, HashToken(s.pepper, accessToken))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	now := s.now().UTC()
	if session.RevokedAt != nil {
		return Session{}, ErrUnauthorized
	}
	if now.After(session.ExpiresAt) {
		return Session{}, ErrSessionExpired
	}
	return session, nil
}

func newID(prefix string) string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return prefix + "_" + hex.EncodeToString(buf)
}
