package sse

import (
	"sync"
	"sync/atomic"
)

// Hub is an in-memory registry of active SSE Clients keyed by a
// user-supplied ID (typically a user ID or session ID).
//
// A single ID may host multiple Clients — same user across browser
// tabs or devices is a common case — and Send fans out to all of them.
// For per-process broadcast use Broadcast.
//
// Hub is safe for concurrent use.
type Hub struct {
	mu      sync.RWMutex
	clients map[string][]*Client
	bufSize int

	// onDrop is invoked once per dropped event when a client's buffer
	// is full. Atomic so it can be swapped without holding mu.
	onDrop atomic.Pointer[func(*Client, Event)]
}

// Client is a registered, active SSE connection.
//
// Create one via Hub.Register, drain it via HubProducer (typical) or
// manually via Recv, remove it via Hub.Unregister or by returning from
// the producer.
//
// Client is safe for concurrent use by the Hub (push) and the producer
// goroutine (recv).
type Client struct {
	// ID is the lookup key the Client was registered under.
	ID string

	// ch buffers events from Hub.Send until the producer reads them.
	ch chan Event

	// done is closed when the Client is unregistered. Producers select
	// on it to exit promptly; push() checks it to short-circuit deliver
	// to a stale Client.
	done chan struct{}

	hub *Hub

	closeOnce sync.Once
	dropped   atomic.Int64
}

// NewHub creates an empty Hub. bufSize is the per-Client channel buffer
// — Send drops events for that Client when the buffer is full. 32-64 is
// a reasonable starting point; larger values trade memory for tolerance
// to bursty producers and slow consumers.
func NewHub(bufSize int) *Hub {
	if bufSize < 1 {
		bufSize = 1
	}
	return &Hub{
		clients: make(map[string][]*Client),
		bufSize: bufSize,
	}
}

// SetOnDrop installs a callback fired exactly once per dropped event.
// Pass nil to remove. Useful for metrics or for unregistering clients
// that fall persistently behind. The callback runs on the Send/Broadcast
// goroutine — keep it short.
func (h *Hub) SetOnDrop(fn func(c *Client, e Event)) {
	if fn == nil {
		h.onDrop.Store(nil)
		return
	}
	h.onDrop.Store(&fn)
}

// Register creates and indexes a new Client under id. The returned
// Client must eventually be Unregistered (HubProducer does this for you).
func (h *Hub) Register(id string) *Client {
	c := &Client{
		ID:   id,
		ch:   make(chan Event, h.bufSize),
		done: make(chan struct{}),
		hub:  h,
	}
	h.mu.Lock()
	h.clients[id] = append(h.clients[id], c)
	h.mu.Unlock()
	return c
}

// Unregister removes c from the Hub. Safe to call multiple times.
// Closing c.done signals waiting consumers (HubProducer) to exit;
// the channel itself is left for GC so concurrent push() calls can't
// panic on send-to-closed-chan.
func (h *Hub) Unregister(c *Client) {
	c.closeOnce.Do(func() {
		close(c.done)

		h.mu.Lock()
		list := h.clients[c.ID]
		for i, x := range list {
			if x == c {
				// Order-preserving removal isn't needed; swap-with-last is fine.
				list[i] = list[len(list)-1]
				list = list[:len(list)-1]
				break
			}
		}
		if len(list) == 0 {
			delete(h.clients, c.ID)
		} else {
			h.clients[c.ID] = list
		}
		h.mu.Unlock()
	})
}

// Send delivers e to every Client registered under id.
// Returns the number of Clients the event reached (delivered) and the
// number whose buffer was full at the time of send (dropped).
//
// If id is unknown both counters are zero.
func (h *Hub) Send(id string, e Event) (delivered, dropped int) {
	h.mu.RLock()
	list := h.clients[id]
	// Snapshot under the lock; do the actual push outside so a slow
	// downstream can't block other Hub operations.
	clients := append([]*Client(nil), list...)
	h.mu.RUnlock()

	for _, c := range clients {
		if c.push(e) {
			delivered++
		} else {
			dropped++
		}
	}
	return
}

// Broadcast delivers e to every active Client across all IDs.
// Returns (delivered, dropped) totals.
func (h *Hub) Broadcast(e Event) (delivered, dropped int) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, list := range h.clients {
		clients = append(clients, list...)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		if c.push(e) {
			delivered++
		} else {
			dropped++
		}
	}
	return
}

// Count returns the totals (active Clients, distinct IDs) — handy for
// /metrics or admin endpoints.
func (h *Hub) Count() (clients, ids int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, list := range h.clients {
		clients += len(list)
	}
	return clients, len(h.clients)
}

// IDs returns a snapshot of currently-registered IDs.
func (h *Hub) IDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

// Has reports whether at least one Client is registered under id.
func (h *Hub) Has(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[id]) > 0
}

// Dropped is the running total of events the Hub had to drop for this
// Client. Read it for per-client backpressure monitoring.
func (c *Client) Dropped() int64 { return c.dropped.Load() }

// push enqueues e onto c.ch without blocking. Returns false if the
// Client is already unregistered or its buffer is full (in the latter
// case dropped++ and OnDrop fires).
//
// The c.done case lets a stale Client short-circuit without touching
// the chan, so Send / Broadcast don't waste buffer slots on dead
// connections waiting for GC.
func (c *Client) push(e Event) bool {
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case c.ch <- e:
		return true
	case <-c.done:
		return false
	default:
		c.dropped.Add(1)
		if fnp := c.hub.onDrop.Load(); fnp != nil {
			(*fnp)(c, e)
		}
		return false
	}
}

// HubProducer returns a Producer that registers a Client under id,
// forwards events from its channel via send, and unregisters on exit.
// Use with Stream / StreamWith:
//
//	r.GET("/events", func(c *rux.Context) {
//	    uid := authUserID(c)
//	    _ = sse.Stream(c, hooks, sse.HubProducer(hub, uid))
//	})
//
// The producer exits cleanly when:
//   - the client disconnects (done from the request context)
//   - hub.Unregister(client) is called from elsewhere
//   - send returns an error (write failure)
func HubProducer(h *Hub, id string) Producer {
	return func(send SendFunc, reqDone <-chan struct{}) error {
		client := h.Register(id)
		defer h.Unregister(client)

		for {
			select {
			case <-reqDone:
				return nil
			case <-client.done:
				return nil
			case e := <-client.ch:
				if err := send(e); err != nil {
					return err
				}
			}
		}
	}
}
