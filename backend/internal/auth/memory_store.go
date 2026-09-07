package auth

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu       sync.Mutex
	users    map[string]User
	sessions map[string]Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:    map[string]User{},
		sessions: map[string]Session{},
	}
}

func (s *MemoryStore) CreateUser(_ context.Context, user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.users {
		if strings.EqualFold(existing.Username, user.Username) {
			return ErrUsernameTaken
		}
	}
	s.users[user.ID] = user
	return nil
}

func (s *MemoryStore) GetUserByUsername(_ context.Context, username string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.users {
		if strings.EqualFold(user.Username, username) {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *MemoryStore) GetUserByID(_ context.Context, id string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *MemoryStore) CreateSession(_ context.Context, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *MemoryStore) GetSessionByAccessHash(_ context.Context, hash string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, session := range s.sessions {
		if session.AccessTokenHash == hash {
			return session, nil
		}
	}
	return Session{}, ErrNotFound
}

func (s *MemoryStore) GetSessionByRefreshHash(_ context.Context, hash string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, session := range s.sessions {
		if session.RefreshTokenHash == hash {
			return session, nil
		}
	}
	return Session{}, ErrNotFound
}

func (s *MemoryStore) UpdateSessionTokens(_ context.Context, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sessions[session.ID]
	if !ok {
		return ErrNotFound
	}
	current.AccessTokenHash = session.AccessTokenHash
	current.RefreshTokenHash = session.RefreshTokenHash
	current.ExpiresAt = session.ExpiresAt
	current.RefreshExpiresAt = session.RefreshExpiresAt
	s.sessions[session.ID] = current
	return nil
}

func (s *MemoryStore) RevokeSession(_ context.Context, sessionID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrNotFound
	}
	session.RevokedAt = &at
	s.sessions[sessionID] = session
	return nil
}
