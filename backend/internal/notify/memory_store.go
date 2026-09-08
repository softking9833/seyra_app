package notify

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu          sync.Mutex
	devices     map[string]Device
	preferences map[string]Preferences
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		devices:     map[string]Device{},
		preferences: map[string]Preferences{},
	}
}

func (s *MemoryStore) UpsertDevice(_ context.Context, device Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, existing := range s.devices {
		if existing.UserID == device.UserID && existing.Token == device.Token {
			delete(s.devices, id)
		}
	}
	s.devices[device.ID] = device
	return nil
}

func (s *MemoryStore) DeleteDevice(_ context.Context, userID, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.devices[deviceID]
	if !ok || item.UserID != userID {
		return ErrNotFound
	}
	delete(s.devices, deviceID)
	return nil
}

func (s *MemoryStore) ListDevices(_ context.Context, userID string) ([]Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Device, 0)
	for _, item := range s.devices {
		if item.UserID == userID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *MemoryStore) GetPreferences(_ context.Context, userID string) (Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prefs, ok := s.preferences[userID]; ok {
		return prefs, nil
	}
	return Preferences{
		UserID:          userID,
		MessagesEnabled: true,
		CallsEnabled:    true,
		ShowPreview:     true,
		UpdatedAt:       time.Time{},
	}, nil
}

func (s *MemoryStore) UpsertPreferences(_ context.Context, prefs Preferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preferences[prefs.UserID] = prefs
	return nil
}
