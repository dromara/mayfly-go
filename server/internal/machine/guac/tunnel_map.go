package guac

import (
	"mayfly-go/pkg/logx"
	"sync"
	"time"
)

/*
LastAccessedTunnel tracks the last time a particular Tunnel was accessed.
This information is not necessary for tunnels associated with WebSocket
connections, as each WebSocket connection has its own read thread which
continuously checks the state of the tunnel and which will automatically
timeout when the underlying stream times out, but the HTTP tunnel has no
such thread. Because the HTTP tunnel requires the stream to be split across
multiple requests, tracking of activity on the tunnel must be performed
independently of the HTTP requests.
*/
type LastAccessedTunnel struct {
	sync.RWMutex
	Tunnel
	lastAccessedTime time.Time
}

func NewLastAccessedTunnel(tunnel Tunnel) (ret LastAccessedTunnel) {
	ret.Tunnel = tunnel
	ret.Access()
	return
}

func (t *LastAccessedTunnel) Access() {
	t.Lock()
	t.lastAccessedTime = time.Now()
	t.Unlock()
}

func (t *LastAccessedTunnel) GetLastAccessedTime() time.Time {
	t.RLock()
	defer t.RUnlock()
	return t.lastAccessedTime
}

/*
TunnelTimeout is the number of seconds to wait between tunnel accesses before timing out.
Note that this will be enforced only within a factor of 2. If a tunnel
is unused, it will take between TUNNEL_TIMEOUT and TUNNEL_TIMEOUT*2
seconds before that tunnel is closed and removed.
*/
const TunnelTimeout = 15 * time.Second

/*
TunnelMap tracks in-use HTTP tunnels, automatically removing
and closing tunnels which have not been used recently. This class is
intended for use only within the Server implementation,
and has no real utility outside that implementation.
*/
type TunnelMap struct {
	sync.RWMutex
	ticker *time.Ticker

	// tunnelTimeout is the maximum amount of time to allow between accesses to any one HTTP tunnel.
	tunnelTimeout time.Duration

	// Map of all tunnels that are using HTTP, indexed by tunnel UUID.
	tunnelMap map[string]*LastAccessedTunnel
}

// NewTunnelMap creates a new TunnelMap and starts the scheduled job with the default timeout.
func NewTunnelMap() *TunnelMap {
	tunnelMap := &TunnelMap{
		ticker:        time.NewTicker(TunnelTimeout),
		tunnelMap:     make(map[string]*LastAccessedTunnel),
		tunnelTimeout: TunnelTimeout,
	}
	go tunnelMap.tunnelTimeoutTask()
	return tunnelMap
}

func (m *TunnelMap) tunnelTimeoutTask() {
	for {
		_, ok := <-m.ticker.C
		if !ok {
			break
		}
		m.tunnelTimeoutTaskRun()
	}
}

func (m *TunnelMap) tunnelTimeoutTaskRun() {
	timeLine := time.Now().Add(0 - m.tunnelTimeout)

	type pair struct {
		uuid   string
		tunnel *LastAccessedTunnel
	}
	var removeIDs []pair

	m.RLock()
	for uuid, tunnel := range m.tunnelMap {
		if tunnel.GetLastAccessedTime().Before(timeLine) {
			removeIDs = append(removeIDs, pair{uuid: uuid, tunnel: tunnel})
		}
	}
	m.RUnlock()

	m.Lock()
	for _, double := range removeIDs {
		logx.Warnf("HTTP tunnel \"%v\" has timed out.", double.uuid)
		delete(m.tunnelMap, double.uuid)

		if double.tunnel != nil {
			err := double.tunnel.Close()
			if err != nil {
				logx.Debugf("Unable to close expired HTTP tunnel. %v", err)
			}
		}
	}
	m.Unlock()
	return
}

// Get returns the Tunnel having the given UUID, wrapped within a LastAccessedTunnel.
func (m *TunnelMap) Get(uuid string) (tunnel *LastAccessedTunnel, ok bool) {
	m.RLock()
	tunnel, ok = m.tunnelMap[uuid]
	m.RUnlock()

	if ok && tunnel != nil {
		tunnel.Access()
	} else {
		ok = false
	}
	return
}

// Add registers that a new connection has been established using HTTP via the given Tunnel.
func (m *TunnelMap) Put(uuid string, tunnel Tunnel) {
	m.Lock()
	one := NewLastAccessedTunnel(tunnel)
	m.tunnelMap[uuid] = &one
	m.Unlock()
}

// Remove removes the Tunnel having the given UUID, if such a tunnel exists. The original tunnel is returned.
func (m *TunnelMap) Remove(uuid string) (*LastAccessedTunnel, bool) {
	m.Lock()
	defer m.Unlock()

	v, ok := m.tunnelMap[uuid]
	if ok {
		delete(m.tunnelMap, uuid)
	}
	return v, ok
}

// Shutdown stops the ticker to free up resources.
func (m *TunnelMap) Shutdown() {
	m.Lock()
	m.ticker.Stop()
	m.Unlock()
}
