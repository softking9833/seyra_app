package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"seyra/backend/internal/auth"
)

func TestHealth(t *testing.T) {
	handler := NewServer(auth.NewService(auth.NewMemoryStore(), "test-session-pepper-value"))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestRegisterLoginSessionLogoutHTTP(t *testing.T) {
	handler := NewServer(auth.NewService(auth.NewMemoryStore(), "test-session-pepper-value"))

	registerBody := bytes.NewBufferString(`{"username":"ada","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", registerBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d body %s", rec.Code, rec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	creds := created["credentials"].(map[string]any)
	access := creds["access_token"].(string)

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"username":"ada","password":"secret"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"ada","password":"nope-secret"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"ada","password":"secret"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	access = created["credentials"].(map[string]any)["access_token"].(string)

	req = httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("session status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status %d", rec.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	handler := NewServer(auth.NewService(auth.NewMemoryStore(), "test-session-pepper-value"))
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"username":"a","password":"secret"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
