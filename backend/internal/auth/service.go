package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
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
	store   Store
	pepper  string
	now     func() time.Time
	loginMu sync.Mutex
	logins  map[string][]time.Time
}

func NewService(store Store, pepper string) *Service {
	return &Service{
		store:  store,
		pepper: pepper,
		now:    time.Now,
		logins: map[string][]time.Time{},
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
	if s.loginLocked(normalized) {
		return IssuedSession{}, ErrRateLimited
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
			s.noteLoginFailure(normalized)
			return IssuedSession{}, ErrInvalidCredentials
		}
		return IssuedSession{}, err
	}
	ok, err := ComparePassword(user.PasswordHash, password)
	if err != nil || !ok {
		s.noteLoginFailure(normalized)
		return IssuedSession{}, ErrInvalidCredentials
	}
	s.clearLoginFailures(normalized)
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

const (
	minSearchQueryRunes = 1
	maxSearchQueryRunes = 32
	maxSearchResults    = 20
)

func NormalizeSearchQuery(query string) (string, error) {
	value := strings.TrimSpace(query)
	if value == "" {
		return "", fmt.Errorf("%w: search query is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(value) < minSearchQueryRunes || utf8.RuneCountInString(value) > maxSearchQueryRunes {
		return "", fmt.Errorf("%w: search query length is invalid", ErrInvalidInput)
	}
	if strings.ContainsAny(value, "%_\\") {
		return "", fmt.Errorf("%w: search query contains invalid characters", ErrInvalidInput)
	}
	for _, r := range value {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return "", fmt.Errorf("%w: search query contains invalid characters", ErrInvalidInput)
		}
	}
	return value, nil
}

type PublicUser struct {
	ID       string
	Username string
}

func (s *Service) SearchUsers(ctx context.Context, accessToken, query string) ([]PublicUser, error) {
	actor, _, err := s.CurrentSession(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	normalized, err := NormalizeSearchQuery(query)
	if err != nil {
		return nil, err
	}
	found, err := s.store.SearchUsers(ctx, normalized, actor.ID, maxSearchResults)
	if err != nil {
		return nil, err
	}
	out := make([]PublicUser, 0, len(found))
	for _, user := range found {
		out = append(out, PublicUser{ID: user.ID, Username: user.Username})
	}
	return out, nil
}

func (s *Service) DeleteAccount(ctx context.Context, accessToken, password string) error {
	user, _, err := s.CurrentSession(ctx, accessToken)
	if err != nil {
		return err
	}
	if err := ValidatePassword(password); err != nil {
		return ErrInvalidCredentials
	}
	ok, err := ComparePassword(user.PasswordHash, password)
	if err != nil || !ok {
		return ErrInvalidCredentials
	}
	return s.store.DeleteUserAndSessions(ctx, user.ID)
}

func (s *Service) DeleteUserRecord(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return ErrNotFound
	}
	return s.store.DeleteUserAndSessions(ctx, userID)
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

func (s *Service) loginLocked(username string) bool {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	cutoff := s.now().Add(-15 * time.Minute)
	var recent []time.Time
	for _, t := range s.logins[username] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	s.logins[username] = recent
	return len(recent) >= 8
}

func (s *Service) noteLoginFailure(username string) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	s.logins[username] = append(s.logins[username], s.now())
}

func (s *Service) clearLoginFailures(username string) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	delete(s.logins, username)
}

func (s *Service) CreateBotUser(ctx context.Context, username string) (User, error) {
	normalized, err := NormalizeUsername(username)
	if err != nil {
		return User{}, err
	}
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	hash, err := HashPassword(hex.EncodeToString(secret))
	if err != nil {
		return User{}, err
	}
	user := User{
		ID: newID("usr"), Username: normalized, PasswordHash: hash, CreatedAt: s.now().UTC(),
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) ListSessions(ctx context.Context, accessToken string) ([]Session, error) {
	_, session, err := s.CurrentSession(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return s.store.ListSessions(ctx, session.UserID)
}

func (s *Service) RevokeSessionID(ctx context.Context, accessToken, sessionID string) error {
	user, current, err := s.CurrentSession(ctx, accessToken)
	if err != nil {
		return err
	}
	sessions, err := s.store.ListSessions(ctx, user.ID)
	if err != nil {
		return err
	}
	found := false
	for _, sess := range sessions {
		if sess.ID == sessionID {
			found = true
			break
		}
	}
	if !found {
		return ErrNotFound
	}
	if sessionID == current.ID {
		return s.store.RevokeSession(ctx, sessionID, s.now().UTC())
	}
	return s.store.RevokeSession(ctx, sessionID, s.now().UTC())
}
