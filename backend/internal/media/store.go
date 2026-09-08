package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrTooLarge          = errors.New("file too large")
	ErrInvalidType       = errors.New("invalid content type")
	ErrInvalidKey        = errors.New("invalid object key")
	ErrNotFound          = errors.New("object not found")
	MaxBytes       int64 = 25 * 1024 * 1024
)

var allowedTypes = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"image/webp":      {},
	"image/gif":       {},
	"application/pdf": {},
	"text/plain":      {},
	"audio/mpeg":      {},
	"audio/mp4":       {},
	"video/mp4":              {},
	"application/octet-stream": {},
}

func AllowedContentType(value string) bool {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(value)), ";")
	_, ok := allowedTypes[base]
	return ok
}

type Store interface {
	Put(ctx context.Context, key, contentType string, r io.Reader, size int64) error
	Open(ctx context.Context, key string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, key string) error
}

type MemoryStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: map[string][]byte{}}
}

func (s *MemoryStore) Put(_ context.Context, key, contentType string, r io.Reader, size int64) error {
	if err := validate(key, contentType, size); err != nil {
		return err
	}
	body, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > MaxBytes {
		return ErrTooLarge
	}
	s.mu.Lock()
	s.data[key] = body
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Open(_ context.Context, key string) (io.ReadCloser, int64, error) {
	s.mu.Lock()
	body, ok := s.data[key]
	s.mu.Unlock()
	if !ok {
		return nil, 0, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(body)), int64(len(body)), nil
}

func (s *MemoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
	return nil
}

type LocalStore struct {
	root string
}

func NewLocalStore(root string) (*LocalStore, error) {
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(".", ".media")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}
	return &LocalStore{root: root}, nil
}

func (s *LocalStore) pathFor(key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	full := filepath.Join(s.root, filepath.FromSlash(key))
	rel, err := filepath.Rel(s.root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", ErrInvalidKey
	}
	return full, nil
}

func (s *LocalStore) Put(_ context.Context, key, contentType string, r io.Reader, size int64) error {
	if err := validate(key, contentType, size); err != nil {
		return err
	}
	full, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return err
	}
	file, err := os.Create(full)
	if err != nil {
		return err
	}
	defer file.Close()
	written, err := io.Copy(file, io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return err
	}
	if written > MaxBytes {
		_ = os.Remove(full)
		return ErrTooLarge
	}
	return nil
}

func (s *LocalStore) Open(_ context.Context, key string) (io.ReadCloser, int64, error) {
	full, err := s.pathFor(key)
	if err != nil {
		return nil, 0, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, err
	}
	file, err := os.Open(full)
	if err != nil {
		return nil, 0, err
	}
	return file, info.Size(), nil
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	full, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func validate(key, contentType string, size int64) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if !AllowedContentType(contentType) {
		return ErrInvalidType
	}
	if size > MaxBytes {
		return ErrTooLarge
	}
	return nil
}

func validateKey(key string) error {
	if key == "" || strings.Contains(key, "..") || strings.ContainsAny(key, `\:`) {
		return ErrInvalidKey
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." {
			return ErrInvalidKey
		}
	}
	return nil
}

func KeyFor(attachmentID, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) > 8 {
		ext = ""
	}
	return fmt.Sprintf("%s%s", attachmentID, ext)
}
