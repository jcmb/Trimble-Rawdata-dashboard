package hub

import (
	"encoding/json"
	"sync"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/model"
)

// Hub broadcasts dashboard events to SSE subscribers.
type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func New() *Hub {
	return &Hub{clients: make(map[chan []byte]struct{})}
}

func (h *Hub) Subscribe() chan []byte {
	ch := make(chan []byte, 8)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *Hub) Publish(ev model.Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- data:
		default:
		}
	}
}

func (h *Hub) PublishSnapshot(snap model.Snapshot) {
	h.Publish(model.Event{Type: "snapshot", Snapshot: snap})
}
