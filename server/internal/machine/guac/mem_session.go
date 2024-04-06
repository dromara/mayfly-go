package guac

import (
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// MemorySessionStore is a simple in-memory store of connected sessions that is used by
// the WebsocketServer to store active sessions.
type MemorySessionStore struct {
	sync.RWMutex
	ConnIds map[uint64]Tunnel
}

// NewMemorySessionStore creates a new store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		ConnIds: map[uint64]Tunnel{},
	}
}

// Get returns a connection by uuid
func (s *MemorySessionStore) Get(id uint64) Tunnel {
	s.RLock()
	defer s.RUnlock()
	return s.ConnIds[id]
}

// Add inserts a new connection by uuid
func (s *MemorySessionStore) Add(id uint64, conn *websocket.Conn, req *http.Request, tunnel Tunnel) {
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
func (s *MemorySessionStore) Delete(id uint64, conn *websocket.Conn, req *http.Request, tunnel Tunnel) {
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
