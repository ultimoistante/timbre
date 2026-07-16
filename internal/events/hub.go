// Package events provides a minimal per-user publish/subscribe hub used to push
// server-sent events (e.g. scan progress) to connected clients.
package events

import "sync"

// Message is a single SSE event.
type Message struct {
	Event string
	Data  []byte
}

// Hub fans out messages to subscribers, partitioned by user id.
type Hub struct {
	mu   sync.RWMutex
	subs map[uint]map[chan Message]struct{}
}

// New creates an empty Hub.
func New() *Hub {
	return &Hub{subs: make(map[uint]map[chan Message]struct{})}
}

// Subscribe registers a new subscriber channel for a user.
func (h *Hub) Subscribe(userID uint) chan Message {
	ch := make(chan Message, 16)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[chan Message]struct{})
	}
	h.subs[userID][ch] = struct{}{}
	return ch
}

// Unsubscribe removes and closes a subscriber channel.
func (h *Hub) Unsubscribe(userID uint, ch chan Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.subs[userID]; ok {
		if _, ok := subs[ch]; ok {
			delete(subs, ch)
			close(ch)
		}
		if len(subs) == 0 {
			delete(h.subs, userID)
		}
	}
}

// Publish delivers a message to all of a user's subscribers without blocking
// the sender. If a subscriber's buffer is full, the oldest queued message is
// evicted to make room — this guarantees the most recent message (crucially,
// a terminal "finished" event) is never lost to a full buffer, at the cost of
// dropping stale intermediate progress ticks.
func (h *Hub) Publish(userID uint, event string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[userID] {
		for {
			select {
			case ch <- Message{Event: event, Data: data}:
			default:
				select {
				case <-ch:
				default:
				}
				continue
			}
			break
		}
	}
}
