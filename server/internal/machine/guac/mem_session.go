package guac

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// MemorySessionStore is a simple in-memory store of connected sessions that is used by
// the WebsocketServer to store active sessions.
type MemorySessionStore struct {
	sync.RWMutex
	ConnIds map[string]Tunnel
}

// NewMemorySessionStore creates a new store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		ConnIds: map[string]Tunnel{},
	}
}

// Get returns a connection by uuid
func (s *MemorySessionStore) Get(id string) Tunnel {
	s.RLock()
	defer s.RUnlock()
	return s.ConnIds[id]
}

// Add inserts a new connection by uuid
func (s *MemorySessionStore) Add(id string, conn *websocket.Conn, req *http.Request, tunnel Tunnel) {
	s.Lock()
	defer s.Unlock()
	n, ok := s.ConnIds[id]
	if !ok {
		s.ConnIds[id] = tunnel
		return
	}
	s.ConnIds[id] = n
	return
}

// Delete removes a connection by uuid
func (s *MemorySessionStore) Delete(id string, conn *websocket.Conn, req *http.Request, tunnel Tunnel) {
	s.Lock()
	defer s.Unlock()
	n, ok := s.ConnIds[id]
	if !ok {
		return
	}
	if n != nil {
		delete(s.ConnIds, id)
		return
	}
	return
}
