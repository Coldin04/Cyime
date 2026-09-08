package editlease

import (
	"sync"

	"github.com/google/uuid"
)

// Hub is a process-local pub/sub registry that lets WebSocket connections
// learn about a document's lease changes the instant they happen, instead of
// waiting for their next renewal poll. It is a courtesy fast-path only: the
// database row remains the single source of truth for who actually holds a
// lease, so a missed or dropped notification never causes incorrect
// behavior — it just falls back to the existing polling latency.
//
// This is intentionally in-memory and per-process. If packages/server is
// ever run as more than one replica behind a load balancer, a claim handled
// by one replica won't reach a subscriber connected to another — at that
// point Hub's Subscribe/Unsubscribe/Publish trio would need to move behind a
// shared bus (e.g. Redis pub/sub) without changing any of its callers.
type Hub struct {
	mu          sync.Mutex
	nextID      int64
	subscribers map[uuid.UUID]map[int64]chan string
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[uuid.UUID]map[int64]chan string)}
}

// Subscribe registers interest in a document's lease-claimed events. The
// returned channel is closed once Unsubscribe is called with the same id;
// callers must always pair Subscribe with a deferred Unsubscribe.
func (h *Hub) Subscribe(documentID uuid.UUID) (int64, <-chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID
	// Small buffer: a claim/takeover is a rare event, so a subscriber only
	// ever needs to be a message or two behind before it catches up.
	ch := make(chan string, 4)
	if h.subscribers[documentID] == nil {
		h.subscribers[documentID] = make(map[int64]chan string)
	}
	h.subscribers[documentID][id] = ch
	return id, ch
}

func (h *Hub) Unsubscribe(documentID uuid.UUID, id int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.subscribers[documentID]
	if !ok {
		return
	}
	if ch, ok := subs[id]; ok {
		close(ch)
		delete(subs, id)
	}
	if len(subs) == 0 {
		delete(h.subscribers, documentID)
	}
}

// Publish notifies every current subscriber of documentID that leaseToken is
// now the active lease token. It never blocks: a subscriber whose buffer is
// already full is skipped rather than stalling the publisher (the caller is
// a claim request in progress), which is safe because the periodic renewal
// poll remains the authoritative fallback.
func (h *Hub) Publish(documentID uuid.UUID, leaseToken string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, ch := range h.subscribers[documentID] {
		select {
		case ch <- leaseToken:
		default:
		}
	}
}

// defaultHub is the process-wide instance every Manager publishes through
// and every WebSocket connection subscribes through. A package-level
// singleton (rather than a field on Manager) keeps it reachable regardless
// of how a Manager was constructed, including the bare struct literals used
// in tests.
var defaultHub = NewHub()

// Subscribe registers interest in documentID's lease-claimed events on the
// default hub. See Hub.Subscribe.
func Subscribe(documentID uuid.UUID) (int64, <-chan string) {
	return defaultHub.Subscribe(documentID)
}

// Unsubscribe removes a subscription registered via Subscribe.
func Unsubscribe(documentID uuid.UUID, id int64) {
	defaultHub.Unsubscribe(documentID, id)
}
