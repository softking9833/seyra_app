package auth

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeUsername(t *testing.T) {
	_, err := NormalizeUsername("ab")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	got, err := NormalizeUsername("  Ada_1  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada_1" {
		t.Fatalf("got %q", got)
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("12345"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if err := ValidatePassword("secret"); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterLoginSessionLogout(t *testing.T) {
	svc := NewService(NewMemoryStore(), "test-session-pepper-value")
	ctx := context.Background()

	issued, err := svc.Register(ctx, "ada", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if issued.AccessToken == "" || issued.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
	if issued.User.PasswordHash != "" && issued.AccessToken == issued.User.PasswordHash {
		t.Fatal("token must not equal password hash")
	}

	_, err = svc.Register(ctx, "ada", "secret")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected username taken, got %v", err)
	}

	_, err = svc.Login(ctx, "ada", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	login, err := svc.Login(ctx, "ada", "secret")
	if err != nil {
		t.Fatal(err)
	}

	user, session, err := svc.CurrentSession(ctx, login.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "ada" || session.ID == "" {
		t.Fatalf("unexpected session user=%s id=%s", user.Username, session.ID)
	}

	if err := svc.Logout(ctx, login.AccessToken); err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.CurrentSession(ctx, login.AccessToken)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized after logout, got %v", err)
	}
}

func TestRefreshRotatesTokens(t *testing.T) {
	svc := NewService(NewMemoryStore(), "test-session-pepper-value")
	ctx := context.Background()
	issued, err := svc.Register(ctx, "lin", "secret1")
	if err != nil {
		t.Fatal(err)
	}
	next, err := svc.Refresh(ctx, issued.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if next.AccessToken == issued.AccessToken || next.RefreshToken == issued.RefreshToken {
		t.Fatal("expected rotated tokens")
	}
	_, _, err = svc.CurrentSession(ctx, issued.AccessToken)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old access token should fail, got %v", err)
	}
	if _, _, err := svc.CurrentSession(ctx, next.AccessToken); err != nil {
		t.Fatal(err)
	}
}
