package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func TestPasswordHashIsArgon2idNotPlaintext(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "secret" || !strings.HasPrefix(hash, "argon2id$") {
		t.Fatalf("expected argon2id encoding, got %q", hash)
	}
	ok, err := ComparePassword(hash, "secret")
	if err != nil || !ok {
		t.Fatal("expected matching password")
	}
}

func TestDeleteAccountRequiresPasswordAndRemovesUser(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, "test-session-pepper-value")
	ctx := context.Background()
	issued, err := svc.Register(ctx, "ada", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteAccount(ctx, issued.AccessToken, "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if _, _, err := svc.CurrentSession(ctx, issued.AccessToken); err != nil {
		t.Fatal("session should remain after failed delete")
	}
	if err := svc.DeleteAccount(ctx, issued.AccessToken, "secret"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.CurrentSession(ctx, issued.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized after delete, got %v", err)
	}
	if _, err := store.GetUserByUsername(ctx, "ada"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected user removed, got %v", err)
	}
	if _, err := svc.Login(ctx, "ada", "secret"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected login to fail after delete, got %v", err)
	}
}

func TestSearchUsers(t *testing.T) {
	svc := NewService(NewMemoryStore(), "test-session-pepper-value")
	ctx := context.Background()
	ada, err := svc.Register(ctx, "ada", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, "lin", "secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, "lisa", "secret"); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SearchUsers(ctx, "", "li"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	if _, err := svc.SearchUsers(ctx, ada.AccessToken, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid query, got %v", err)
	}
	if _, err := svc.SearchUsers(ctx, ada.AccessToken, "li%"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected wildcard rejection, got %v", err)
	}

	found, err := svc.SearchUsers(ctx, ada.AccessToken, "li")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("expected lin and lisa, got %+v", found)
	}
	for _, user := range found {
		if user.Username == "ada" || strings.Contains(fmt.Sprintf("%+v", user), "Password") {
			t.Fatalf("search leaked self or secrets: %+v", user)
		}
		if user.ID == "" || user.Username == "" {
			t.Fatalf("missing public fields %+v", user)
		}
	}
}

func TestUpdateProfileAndUsername(t *testing.T) {
	svc := NewService(NewMemoryStore(), "test-session-pepper-value")
	ctx := context.Background()
	issued, err := svc.Register(ctx, "ada", "secret")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateProfile(ctx, issued.AccessToken, "Ada Lovelace", "Builder")
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != "Ada Lovelace" || updated.Bio != "Builder" {
		t.Fatalf("profile %+v", updated)
	}
	renamed, err := svc.ChangeUsername(ctx, issued.AccessToken, "ada_prime")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Username != "ada_prime" {
		t.Fatalf("username %q", renamed.Username)
	}
	_, err = svc.Register(ctx, "lin", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ChangeUsername(ctx, issued.AccessToken, "lin"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected taken, got %v", err)
	}
	other, err := svc.Login(ctx, "ada_prime", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RevokeOtherSessions(ctx, other.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.CurrentSession(ctx, issued.AccessToken); !errors.Is(err, ErrUnauthorized) && !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected revoked first session, got %v", err)
	}
}
