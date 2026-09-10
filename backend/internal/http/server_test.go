package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"seyra/backend/internal/auth"
	"seyra/backend/internal/chat"
	"seyra/backend/internal/notify"
)

func testHandler() http.Handler {
	store := auth.NewMemoryStore()
	hub := chat.NewHub()
	authService := auth.NewService(store, "test-session-pepper-value")
	chatService := chat.NewService(chat.NewMemoryStore(), chat.NewAuthDirectory(store), hub)
	alerts := notify.NewService(notify.NewMemoryStore(), &notify.RecordingSender{})
	return NewServer(authService, chatService, hub, alerts)
}

func TestHealth(t *testing.T) {
	handler := testHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestRegisterLoginSessionLogoutHTTP(t *testing.T) {
	handler := testHandler()

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

func TestProfileMeAndDeleteAccountHTTP(t *testing.T) {
	handler := testHandler()

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"username":"ada","password":"secret"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d body %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	access := created["credentials"].(map[string]any)["access_token"].(string)

	req = httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("me status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "password") || strings.Contains(body, "access_token") || strings.Contains(body, "refresh_token") {
		t.Fatalf("profile leaked secrets: %s", body)
	}
	var profile map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile["username"] != "ada" || profile["id"] == nil || profile["created_at"] == nil {
		t.Fatalf("unexpected profile %v", profile)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/account/delete", bytes.NewBufferString(`{"password":"nope-secret"}`))
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/account/delete", bytes.NewBufferString(`{"password":"secret"}`))
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status %d body %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after delete, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"ada","password":"secret"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 login after delete, got %d", rec.Code)
	}
}

func TestRegisterValidation(t *testing.T) {
	handler := testHandler()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"username":"a","password":"secret"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func registerUser(t *testing.T, handler http.Handler, username string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"username":"`+username+`","password":"secret"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s status %d body %s", username, rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	return created["credentials"].(map[string]any)["access_token"].(string)
}

func TestChatHTTPAndRealtime(t *testing.T) {
	handler := testHandler()
	ada := registerUser(t, handler, "ada")
	lin := registerUser(t, handler, "lin")

	req := httptest.NewRequest(http.MethodPost, "/v1/chats", bytes.NewBufferString(`{"username":"ghost"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/chats", bytes.NewBufferString(`{"username":"ada"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 self chat, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/chats", bytes.NewBufferString(`{"username":"lin"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create chat %d %s", rec.Code, rec.Body.String())
	}
	var chatBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &chatBody); err != nil {
		t.Fatal(err)
	}
	chatID := chatBody["id"].(string)

	req = httptest.NewRequest(http.MethodPost, "/v1/chats", bytes.NewBufferString(`{"username":"lin"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("duplicate create %d", rec.Code)
	}
	var again map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &again)
	if again["id"] != chatID {
		t.Fatalf("expected same chat id")
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats/"+chatID+"/messages", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	stranger := registerUser(t, handler, "moe")
	req = httptest.NewRequest(http.MethodGet, "/v1/chats/"+chatID+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+stranger)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/realtime"

	if _, resp, err := websocket.DefaultDialer.Dial(wsURL, nil); err == nil {
		t.Fatal("expected websocket auth failure")
	} else if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 upgrade, got err=%v resp=%v", err, resp)
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+lin)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("ws dial: %v status %v", err, resp)
	}
	defer conn.Close()

	req = httptest.NewRequest(http.MethodPost, "/v1/chats/"+chatID+"/messages", bytes.NewBufferString(`{"body":"Hello from Seyra"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send %d %s", rec.Code, rec.Body.String())
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var event map[string]any
	for {
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatalf("read event: %v", err)
		}
		if event["type"] == "realtime.connected" || event["type"] == "receipt.updated" {
			continue
		}
		break
	}
	if event["type"] != "message.created" {
		t.Fatalf("unexpected event %v", event)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats", nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"unread_count":1`) {
		t.Fatalf("expected unread for lin, got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats/"+chatID+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list messages %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Hello from Seyra") {
		t.Fatalf("missing persisted body %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats", nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Hello from Seyra") {
		t.Fatalf("chat list %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats/"+chatID+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("lin read %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats", nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `"unread_count":1`) {
		t.Fatalf("expected unread cleared after read, got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/chats/"+chatID+"/messages", bytes.NewBufferString(`{"body":"second"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("second send %d %s", rec.Code, rec.Body.String())
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var secondCreated map[string]any
	for {
		if err := conn.ReadJSON(&secondCreated); err != nil {
			t.Fatalf("read second created: %v", err)
		}
		if secondCreated["type"] == "receipt.updated" || secondCreated["type"] == "realtime.connected" {
			continue
		}
		break
	}
	if secondCreated["type"] != "message.created" {
		t.Fatalf("expected second created, got %v", secondCreated)
	}
	var second map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	secondID := second["id"].(string)

	req = httptest.NewRequest(http.MethodDelete, "/v1/chats/"+chatID+"/messages/"+secondID, nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 deleting others message, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/v1/chats/"+chatID+"/messages/"+secondID, nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete %d %s", rec.Code, rec.Body.String())
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var deletedEvent map[string]any
	if err := conn.ReadJSON(&deletedEvent); err != nil {
		t.Fatalf("read deleted event: %v", err)
	}
	if deletedEvent["type"] != "message.deleted" {
		t.Fatalf("unexpected deleted event %v", deletedEvent)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats/"+chatID+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "second") {
		t.Fatalf("soft delete should hide body, got %s", rec.Body.String())
	}

	mayaTok := registerUser(t, handler, "maya")
	req = httptest.NewRequest(http.MethodPost, "/v1/chats/groups", bytes.NewBufferString(`{"title":"Design Team","usernames":["lin","maya"]}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"kind":"group"`) {
		t.Fatalf("create group %d %s", rec.Code, rec.Body.String())
	}
	var group map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &group); err != nil {
		t.Fatal(err)
	}
	groupID := group["id"].(string)
	req = httptest.NewRequest(http.MethodGet, "/v1/chats/"+groupID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"role":"owner"`) {
		t.Fatalf("list members %d %s", rec.Code, rec.Body.String())
	}
	var memberList struct {
		Members []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"members"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &memberList); err != nil {
		t.Fatal(err)
	}
	var adaID, linID, mayaID string
	for _, m := range memberList.Members {
		switch m.Username {
		case "ada":
			adaID = m.ID
		case "lin":
			linID = m.ID
		case "maya":
			mayaID = m.ID
		}
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/chats/"+groupID+"/members/"+linID+"/role", bytes.NewBufferString(`{"role":"admin"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("promote %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodDelete, "/v1/chats/"+groupID+"/members/"+adaID, nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected owner protected, got %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/chats/"+groupID+"/leave", nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("owner cannot leave, got %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/chats/"+groupID+"/leave", nil)
	req.Header.Set("Authorization", "Bearer "+mayaTok)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("member leave %d %s", rec.Code, rec.Body.String())
	}

	_ = mayaID
	req = httptest.NewRequest(http.MethodPost, "/v1/chats/channels", bytes.NewBufferString(`{"title":"Seyra News","usernames":["lin"]}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"kind":"channel"`) {
		t.Fatalf("create channel %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/chats", nil)
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Design Team") || !strings.Contains(rec.Body.String(), "Seyra News") {
		t.Fatalf("room list %d %s", rec.Code, rec.Body.String())
	}
}

func TestUserSearchHTTP(t *testing.T) {
	handler := testHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/search?q=li", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	ada := registerUser(t, handler, "ada")
	_ = registerUser(t, handler, "lin")
	_ = registerUser(t, handler, "lisa")
	for i := 0; i < 25; i++ {
		_ = registerUser(t, handler, fmt.Sprintf("li%x", i+10))
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/users/search?q=", nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 empty query, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/users/search?q=li", nil)
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "password") || strings.Contains(body, "token") {
		t.Fatalf("search leaked secrets: %s", body)
	}
	var parsed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	users := parsed["users"].([]any)
	if len(users) != 20 {
		t.Fatalf("expected search limit 20, got %d", len(users))
	}
	for _, raw := range users {
		item := raw.(map[string]any)
		if item["username"] == "ada" {
			t.Fatal("search should not return the caller")
		}
	}
}

func TestNotificationDevicesHTTP(t *testing.T) {
	handler := testHandler()
	req := httptest.NewRequest(http.MethodPost, "/v1/notifications/devices", bytes.NewBufferString(`{"platform":"android","token":"device-secret"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	ada := registerUser(t, handler, "ada")
	lin := registerUser(t, handler, "lin")
	req = httptest.NewRequest(http.MethodPost, "/v1/notifications/devices", bytes.NewBufferString(`{"platform":"android","token":"device-secret"}`))
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register device %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "device-secret") {
		t.Fatal("device token must not be returned")
	}

	req = httptest.NewRequest(http.MethodPut, "/v1/notifications/preferences", bytes.NewBufferString(`{"messages_enabled":true,"calls_enabled":true,"show_preview":false}`))
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"show_preview":false`) {
		t.Fatalf("prefs %s", rec.Body.String())
	}

	store := auth.NewMemoryStore()
	hub := chat.NewHub()
	authService := auth.NewService(store, "test-session-pepper-value")
	chatService := chat.NewService(chat.NewMemoryStore(), chat.NewAuthDirectory(store), hub)
	sender := &notify.RecordingSender{}
	alerts := notify.NewService(notify.NewMemoryStore(), sender)
	wired := NewServer(authService, chatService, hub, alerts)
	ada = registerUser(t, wired, "ada")
	lin = registerUser(t, wired, "lin")
	req = httptest.NewRequest(http.MethodPost, "/v1/notifications/devices", bytes.NewBufferString(`{"platform":"android","token":"lin-token"}`))
	req.Header.Set("Authorization", "Bearer "+lin)
	rec = httptest.NewRecorder()
	wired.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("lin device %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/chats", bytes.NewBufferString(`{"username":"lin"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	wired.ServeHTTP(rec, req)
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	chatID := created["id"].(string)
	req = httptest.NewRequest(http.MethodPost, "/v1/chats/"+chatID+"/messages", bytes.NewBufferString(`{"body":"ping"}`))
	req.Header.Set("Authorization", "Bearer "+ada)
	rec = httptest.NewRecorder()
	wired.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send %d %s", rec.Code, rec.Body.String())
	}
	if sender.Count != 1 {
		t.Fatalf("expected push to lin, got %d", sender.Count)
	}
}

func TestProfileUsernamePrivacyAndSessionsHTTP(t *testing.T) {
	handler := testHandler()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"username":"ada","password":"secret"}`))
	req.Header.Set("User-Agent", "SeyraTest/1.0")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	access := created["credentials"].(map[string]any)["access_token"].(string)

	req = httptest.NewRequest(http.MethodPatch, "/v1/users/me", bytes.NewBufferString(`{"display_name":"Ada L","bio":"hi"}`))
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch me %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/v1/users/me", bytes.NewBufferString(`{"display_name":"Ada Put","bio":"put"}`))
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put me %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var me map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &me)
	if me["display_name"] != "Ada Put" || me["bio"] != "put" {
		t.Fatalf("me %+v", me)
	}

	req = httptest.NewRequest(http.MethodPatch, "/v1/users/me/username", bytes.NewBufferString(`{"username":"ada2"}`))
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("username %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/v1/privacy", bytes.NewBufferString(`{"photo_visible":false,"last_seen_visible":false}`))
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("privacy %d %s", rec.Code, rec.Body.String())
	}
	var priv map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &priv)
	if priv["photo_visible"] != false || priv["last_seen_visible"] != false {
		t.Fatalf("privacy %+v", priv)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"username":"ada2","password":"secret"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var second map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &second)
	secondAccess := second["credentials"].(map[string]any)["access_token"].(string)

	req = httptest.NewRequest(http.MethodGet, "/v1/auth/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+secondAccess)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var sessions map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &sessions)
	list := sessions["sessions"].([]any)
	if len(list) < 2 {
		t.Fatalf("expected 2 sessions, got %+v", sessions)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/auth/sessions/others", nil)
	req.Header.Set("Authorization", "Bearer "+secondAccess)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("revoke others %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/auth/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("old session should be revoked, got %d", rec.Code)
	}
}

func TestAvatarUploadLargerThanJSONLimit(t *testing.T) {
	handler := testHandler()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"username":"ada","password":"secret"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	access := created["credentials"].(map[string]any)["access_token"].(string)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="avatar.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 20*1024)
	payload[0], payload[1], payload[2] = 0xff, 0xd8, 0xff
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/users/me/avatar", &body)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("avatar %d %s", rec.Code, rec.Body.String())
	}
	var me map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me["has_avatar"] != true {
		t.Fatalf("expected avatar, got %+v", me)
	}
}
