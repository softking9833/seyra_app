package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const seedPassword = "seyra123"

func main() {
	base := os.Getenv("SEYRA_API_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	client := &http.Client{Timeout: 15 * time.Second}

	users := []string{"ada", "lin", "maya", "jordan", "noah", "sam"}
	tokens := map[string]string{}
	for _, username := range users {
		token, err := ensureUser(client, base, username, seedPassword)
		if err != nil {
			fmt.Fprintf(os.Stderr, "user %s: %v\n", username, err)
			os.Exit(1)
		}
		tokens[username] = token
	}

	existing := map[string]struct{}{}
	if ronanToken, err := login(client, base, "ronan", os.Getenv("SEYRA_SEED_RONAN_PASSWORD")); err == nil {
		tokens["ronan"] = ronanToken
		existing["ronan"] = struct{}{}
		fmt.Println("included existing user ronan")
	}

	roomPeers := []string{"lin", "maya", "jordan", "noah", "sam"}
	if _, ok := tokens["ronan"]; ok {
		roomPeers = append(roomPeers, "ronan")
	}

	directs := []struct {
		from, to, body string
	}{
		{"ada", "lin", "Did you get the encrypted files?"},
		{"lin", "ada", "Yes — stored only on device."},
		{"ada", "maya", "See you at 6. I'll bring the keys."},
		{"ada", "jordan", "Voice notes can wait. Text is enough for now."},
		{"ada", "noah", "Thanks — received and deleted on my side."},
	}
	if _, ok := tokens["ronan"]; ok {
		directs = append(directs,
			struct{ from, to, body string }{"lin", "ronan", "Hey Ronan — this is a seeded DM for testing."},
			struct{ from, to, body string }{"maya", "ronan", "Ping me in Design Team when you are in."},
		)
	}

	for _, item := range directs {
		chatID, err := createDirect(client, base, tokens[item.from], item.to)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dm %s -> %s: %v\n", item.from, item.to, err)
			os.Exit(1)
		}
		if err := sendMessage(client, base, tokens[item.from], chatID, item.body); err != nil {
			fmt.Fprintf(os.Stderr, "message %s -> %s: %v\n", item.from, item.to, err)
			os.Exit(1)
		}
	}

	rooms := []struct {
		kind, title, from, body string
		usernames               []string
	}{
		{kind: "groups", title: "Design Team", from: "ada", usernames: roomPeers, body: "Let's ship the new header tonight."},
		{kind: "groups", title: "Weekend Hikers", from: "maya", usernames: []string{"ada", "lin", "sam", "jordan"}, body: "Trail photos are in the album."},
		{kind: "channels", title: "Seyra News", from: "ada", usernames: roomPeers, body: "Privacy update is live for everyone."},
		{kind: "channels", title: "Security Tips", from: "lin", usernames: []string{"ada", "maya", "noah"}, body: "How to lock down your account in 2 minutes."},
	}
	if _, ok := tokens["ronan"]; ok {
		rooms[1].usernames = append(rooms[1].usernames, "ronan")
		rooms[3].usernames = append(rooms[3].usernames, "ronan")
	}

	for _, room := range rooms {
		id, err := createRoom(client, base, tokens[room.from], room.kind, room.title, room.usernames)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: %v\n", room.kind, room.title, err)
			os.Exit(1)
		}
		if err := sendMessage(client, base, tokens[room.from], id, room.body); err != nil {
			fmt.Fprintf(os.Stderr, "message %s: %v\n", room.title, err)
			os.Exit(1)
		}
	}

	fmt.Println("seeded users: " + strings.Join(users, ", "))
	fmt.Println("password for seeded users: " + seedPassword)
	fmt.Println("groups: Design Team, Weekend Hikers")
	fmt.Println("channels: Seyra News, Security Tips")
}

func ensureUser(client *http.Client, base, username, password string) (string, error) {
	token, err := register(client, base, username, password)
	if err == nil {
		return token, nil
	}
	if !strings.Contains(err.Error(), "username_taken") {
		return "", err
	}
	return login(client, base, username, password)
}

func register(client *http.Client, base, username, password string) (string, error) {
	status, body, err := postJSON(client, base+"/v1/auth/register", "", map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	if status == http.StatusConflict {
		return "", fmt.Errorf("username_taken")
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("register %s: %d %s", username, status, body)
	}
	return accessToken(body)
}

func login(client *http.Client, base, username, password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("no password")
	}
	status, body, err := postJSON(client, base+"/v1/auth/login", "", map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("login %s: %d", username, status)
	}
	return accessToken(body)
}

func createDirect(client *http.Client, base, token, username string) (string, error) {
	status, body, err := postJSON(client, base+"/v1/chats", token, map[string]string{
		"username": username,
	})
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("status %d %s", status, body)
	}
	return objectID(body)
}

func createRoom(client *http.Client, base, token, kind, title string, usernames []string) (string, error) {
	status, body, err := postJSON(client, base+"/v1/chats/"+kind, token, map[string]any{
		"title":     title,
		"usernames": usernames,
	})
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated {
		return "", fmt.Errorf("status %d %s", status, body)
	}
	return objectID(body)
}

func sendMessage(client *http.Client, base, token, chatID, bodyText string) error {
	status, body, err := postJSON(client, base+"/v1/chats/"+chatID+"/messages", token, map[string]string{
		"body": bodyText,
	})
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("status %d %s", status, body)
	}
	return nil
}

func postJSON(client *http.Client, url, token string, payload any) (int, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

func accessToken(body []byte) (string, error) {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	creds, _ := parsed["credentials"].(map[string]any)
	token, _ := creds["access_token"].(string)
	if token == "" {
		return "", fmt.Errorf("missing access token")
	}
	return token, nil
}

func objectID(body []byte) (string, error) {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	id, _ := parsed["id"].(string)
	if id == "" {
		return "", fmt.Errorf("missing id")
	}
	return id, nil
}
