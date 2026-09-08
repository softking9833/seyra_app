package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrivacyBlockInviteAndBotHTTP(t *testing.T) {
	handler := testHandler()
	ada := accessToken(t, handler, "ada", "secret1")
	lin := accessToken(t, handler, "lin", "secret1")

	req := httptest.NewRequest(http.MethodPut, "/v1/privacy", bytes.NewBufferString(`{"profile_visible":false}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("privacy %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/blocks", bytes.NewBufferString(`{"username":"lin"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("block %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/chats", bytes.NewBufferString(`{"username":"lin"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("blocked chat expected 403 got %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/chats/groups", bytes.NewBufferString(`{"title":"Team","usernames":["lin"]}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("group %d %s", rec.Code, rec.Body.String())
	}
	var group map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &group); err != nil {
		t.Fatal(err)
	}
	groupID := group["id"].(string)

	req = httptest.NewRequest(http.MethodPost, "/v1/chats/"+groupID+"/invites", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("invite %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/bots", bytes.NewBufferString(`{"username":"echo_bot"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("bot %d %s", rec.Code, rec.Body.String())
	}
	var bot map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &bot); err != nil {
		t.Fatal(err)
	}
	botID := bot["id"].(string)
	token := bot["token"].(string)
	if token == "" {
		t.Fatal("missing bot token")
	}

	grant := `{"conversation_id":"` + groupID + `","can_send":true,"can_read":true}`
	req = httptest.NewRequest(http.MethodPost, "/v1/bots/"+botID+"/grants", bytes.NewBufferString(grant))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("grant %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/chats/"+groupID+"/messages", bytes.NewBufferString(`{"body":"/ping"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ping %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/calls/ice", nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ice %d %s", rec.Code, rec.Body.String())
	}

	_ = lin
}

func TestHiddenProfileAndBotCannotRead(t *testing.T) {
	handler := testHandler()
	ada := accessToken(t, handler, "ada2", "secret1")
	lin := accessToken(t, handler, "lin2", "secret1")

	req := httptest.NewRequest(http.MethodPut, "/v1/privacy", bytes.NewBufferString(`{"profile_visible":false}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("privacy %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/users/search?q=ada", nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"ada2"`) {
		t.Fatalf("hidden profile leaked: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/chats/groups", bytes.NewBufferString(`{"title":"Ops","usernames":["lin2"]}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("group %d %s", rec.Code, rec.Body.String())
	}
	var group map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &group); err != nil {
		t.Fatal(err)
	}
	groupID := group["id"].(string)

	req = httptest.NewRequest(http.MethodPost, "/v1/bots", bytes.NewBufferString(`{"username":"reader_bot"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("bot %d %s", rec.Code, rec.Body.String())
	}
	var bot map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &bot); err != nil {
		t.Fatal(err)
	}
	botID := bot["id"].(string)
	token := bot["token"].(string)

	grant := `{"conversation_id":"` + groupID + `","can_send":true,"can_read":false}`
	req = httptest.NewRequest(http.MethodPost, "/v1/bots/"+botID+"/grants", bytes.NewBufferString(grant))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("grant %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats/"+groupID+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("bot read expected 403 got %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/bots", nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list bots %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("bot token leaked on list: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/stickers", nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "wave") {
		t.Fatalf("stickers %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats/"+groupID+"/attachments", nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("gallery %d %s", rec.Code, rec.Body.String())
	}
}

func accessToken(t *testing.T, handler http.Handler, username, password string) string {
	t.Helper()
	body := `{"username":"` + username + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s %d %s", username, rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	return created["credentials"].(map[string]any)["access_token"].(string)
}
