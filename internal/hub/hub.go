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
	_, ok := h.clients[ch]
	if ok {
		delete(h.clients, ch)
	}
	h.mu.Unlock()
	if ok {
		close(ch)
	}
}

func (h *Hub) Publish(ev model.Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	chans := make([]chan []byte, 0, len(h.clients))
	for ch := range h.clients {
		chans = append(chans, ch)
	}
	h.mu.RUnlock()

	for _, ch := range chans {
		h.trySend(ch, data)
	}
}

func (h *Hub) trySend(ch chan []byte, data []byte) {
	defer func() { _ = recover() }()
	select {
	case ch <- data:
	default:
	}
}

func (h *Hub) PublishSnapshot(snap model.Snapshot) {
	h.Publish(model.Event{Type: "snapshot", Snapshot: snap})
}
