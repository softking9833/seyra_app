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

func (s *MemoryStore) SearchUsers(_ context.Context, query, excludeUserID string, limit int) ([]User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := strings.ToLower(query)
	out := make([]User, 0)
	for _, user := range s.users {
		if user.ID == excludeUserID {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(user.Username), prefix) {
			continue
		}
		out = append(out, User{ID: user.ID, Username: user.Username, CreatedAt: user.CreatedAt})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
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

func (s *MemoryStore) DeleteUserAndSessions(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return ErrNotFound
	}
	for id, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, id)
		}
	}
	delete(s.users, userID)
	return nil
}

func (s *MemoryStore) SetSessionUserAgent(_ context.Context, sessionID, userAgent string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrNotFound
	}
	session.UserAgent = userAgent
	s.sessions[sessionID] = session
	return nil
}

func (s *MemoryStore) UpdateUserProfile(_ context.Context, userID, displayName, bio string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return ErrNotFound
	}
	user.DisplayName = displayName
	user.Bio = bio
	s.users[userID] = user
	return nil
}

func (s *MemoryStore) UpdateUsername(_ context.Context, userID, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return ErrNotFound
	}
	for _, existing := range s.users {
		if existing.ID != userID && strings.EqualFold(existing.Username, username) {
			return ErrUsernameTaken
		}
	}
	user.Username = username
	s.users[userID] = user
	return nil
}

func (s *MemoryStore) SetAvatarKey(_ context.Context, userID, avatarKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return ErrNotFound
	}
	user.AvatarKey = avatarKey
	s.users[userID] = user
	return nil
}

func (s *MemoryStore) ListSessions(_ context.Context, userID string) ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Session
	for _, session := range s.sessions {
		if session.UserID == userID {
			out = append(out, session)
		}
	}
	return out, nil
}
