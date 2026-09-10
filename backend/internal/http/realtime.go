package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"seyra/backend/internal/chat"
)

var realtimeUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) realtime(w http.ResponseWriter, r *http.Request) {
	token := realtimeAccessToken(r)
	user, _, err := s.auth.CurrentSession(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	upgradeHeader := http.Header{}
	if proto := selectedSeyraProtocol(r); proto != "" {
		upgradeHeader.Set("Sec-WebSocket-Protocol", proto)
	}
	conn, err := realtimeUpgrader.Upgrade(w, r, upgradeHeader)
	if err != nil {
		log.Printf("realtime upgrade failed")
		return
	}
	defer conn.Close()

	events, cancel := s.hub.Subscribe(user.ID)
	defer cancel()

	conn.SetReadLimit(64 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if _, _, err := s.auth.CurrentSession(r.Context(), token); err != nil {
				return
			}
			var event chat.Event
			if err := json.Unmarshal(data, &event); err != nil {
				continue
			}
			if event.Type == "" {
				continue
			}
			_ = s.chat.HandleRealtime(r.Context(), user.ID, event)
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	_ = conn.WriteJSON(chat.Event{Type: "realtime.connected", Payload: map[string]any{}})

	for {
		select {
		case <-done:
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			if _, _, err := s.auth.CurrentSession(r.Context(), token); err != nil {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

const seyraWSProtocolPrefix = "seyra."

func realtimeAccessToken(r *http.Request) string {
	if token := bearerToken(r); token != "" {
		return token
	}
	return tokenFromWebsocketProtocol(r)
}

func selectedSeyraProtocol(r *http.Request) string {
	for _, proto := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		proto = strings.TrimSpace(proto)
		if strings.HasPrefix(proto, seyraWSProtocolPrefix) {
			return proto
		}
	}
	return ""
}

func tokenFromWebsocketProtocol(r *http.Request) string {
	proto := selectedSeyraProtocol(r)
	if proto == "" {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(proto, seyraWSProtocolPrefix))
	if err != nil || len(raw) == 0 {
		return ""
	}
	return string(raw)
}
