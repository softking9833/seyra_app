package auth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

func NormalizeUsername(username string) (string, error) {
	value := strings.TrimSpace(username)
	if value == "" {
		return "", fmt.Errorf("%w: username is required", ErrInvalidInput)
	}
	if !usernamePattern.MatchString(value) {
		return "", fmt.Errorf("%w: username must be 3-32 letters, numbers, or underscores", ErrInvalidInput)
	}
	return value, nil
}

func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < 6 {
		return fmt.Errorf("%w: password must be at least 6 characters", ErrInvalidInput)
	}
	if length > 128 {
		return fmt.Errorf("%w: password is too long", ErrInvalidInput)
	}
	return nil
}

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrUsernameTaken      = errors.New("username taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session expired")
	ErrNotFound           = errors.New("not found")
	ErrRateLimited        = errors.New("rate limited")
)
