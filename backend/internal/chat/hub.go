package chat

import "sync"

// Hub is an in-process fan-out. A Redis pub/sub Publisher can replace it later
// without changing Service call sites.
type Hub struct {
	mu   sync.Mutex
	next int
	subs map[int]subscription
}

type subscription struct {
	userID string
	ch     chan Event
}

func NewHub() *Hub {
	return &Hub{subs: map[int]subscription{}}
}

func (h *Hub) Subscribe(userID string) (<-chan Event, func()) {
	ch := make(chan Event, 64)
	h.mu.Lock()
	id := h.next
	h.next++
	h.subs[id] = subscription{userID: userID, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		if sub, ok := h.subs[id]; ok {
			delete(h.subs, id)
			close(sub.ch)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) Publish(userIDs []string, event Event) {
	wanted := map[string]struct{}{}
	for _, id := range userIDs {
		wanted[id] = struct{}{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, sub := range h.subs {
		if _, ok := wanted[sub.userID]; !ok {
			continue
		}
		select {
		case sub.ch <- event:
		default:
		}
	}
}
