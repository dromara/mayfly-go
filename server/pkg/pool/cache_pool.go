package pool

import (
	"context"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"sync"
	"time"
)

var CachePoolDefaultConfig = PoolConfig{
	MaxConns:            1,
	IdleTimeout:         60 * time.Minute,
	WaitTimeout:         10 * time.Second,
	HealthCheckInterval: 10 * time.Minute,
}

type cacheEntry[T Conn] struct {
	conn       T
	lastActive time.Time
}

func (e *cacheEntry[T]) Close() {
	if err := e.conn.Close(); err != nil {
		logx.Errorf("cache pool - closing connection error: %v", err)
	}
}

type CachePool[T Conn] struct {
	factory func() (T, error)
	mu      sync.RWMutex
	cache   map[string]*cacheEntry[T] // 使用字符串键的缓存
	config  PoolConfig
	closeCh chan struct{}
	closed  bool
}

func NewCachePool[T Conn](factory func() (T, error), opts ...Option) *CachePool[T] {
	config := CachePoolDefaultConfig
	for _, opt := range opts {
		opt(&config)
	}

	p := &CachePool[T]{
		factory: factory,
		cache:   make(map[string]*cacheEntry[T]),
		config:  config,
		closeCh: make(chan struct{}),
	}

	go p.backgroundMaintenance()
	return p
}

// Get 获取连接（自动创建或复用缓存连接）
func (p *CachePool[T]) Get(ctx context.Context) (T, error) {
	var zero T

	// 先尝试加读锁，仅用于查找可用连接
	p.mu.RLock()
	for _, entry := range p.cache {
		if time.Since(entry.lastActive) <= p.config.IdleTimeout {
			p.mu.RUnlock() // 找到后释放读锁
			return entry.conn, nil
		}
	}
	p.mu.RUnlock()

	// 没有找到可用连接，升级为写锁进行清理和创建
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return zero, ErrPoolClosed
	}

	// 再次检查是否已有可用连接（防止并发创建）
	for key, entry := range p.cache {
		if time.Since(entry.lastActive) <= p.config.IdleTimeout {
			entry.lastActive = time.Now()
			return entry.conn, nil
		}
		// 清理超时连接
		entry.Close()
		delete(p.cache, key)
	}

	// 创建新连接
	conn, err := p.factory()
	if err != nil {
		return zero, err
	}

	p.cache[generateCacheKey()] = &cacheEntry[T]{
		conn:       conn,
		lastActive: time.Now(),
	}

	return conn, nil
}

// Put 将连接放回缓存
func (p *CachePool[T]) Put(conn T) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return conn.Close()
	}

	p.cache[generateCacheKey()] = &cacheEntry[T]{
		conn:       conn,
		lastActive: time.Now(),
	}

	// 如果超出最大连接数，清理最久未使用的
	if len(p.cache) > p.config.MaxConns {
		p.removeOldest()
	}

	return nil
}

// 移除最久未使用的连接
func (p *CachePool[T]) removeOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range p.cache {
		if oldestKey == "" || entry.lastActive.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastActive
		}
	}

	if oldestKey != "" {
		p.cache[oldestKey].conn.Close()
		delete(p.cache, oldestKey)
	}
}

// Close 关闭连接池
func (p *CachePool[T]) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.closeCh)

	for _, entry := range p.cache {
		entry.Close()
	}

	// 触发关闭回调
	if p.config.OnPoolClose != nil {
		p.config.OnPoolClose()
	}

	p.cache = make(map[string]*cacheEntry[T])
}

// Resize 动态调整大小
func (p *CachePool[T]) Resize(newSize int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed || newSize == p.config.MaxConns {
		return
	}

	p.config.MaxConns = newSize

	// 如果新大小小于当前缓存数量，清理多余的连接
	for len(p.cache) > newSize {
		p.removeOldest()
	}
}

// Stats 获取统计信息
func (p *CachePool[T]) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		TotalConns: int32(len(p.cache)),
	}
}

// 后台维护协程
func (p *CachePool[T]) backgroundMaintenance() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupIdle()
		case <-p.closeCh:
			return
		}
	}
}

// 清理闲置超时的连接
func (p *CachePool[T]) cleanupIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	cutoff := time.Now().Add(-p.config.IdleTimeout)
	for key, entry := range p.cache {
		if entry.lastActive.Before(cutoff) {
			entry.Close()
			delete(p.cache, key)
		}
	}
}

// 生成缓存键
func generateCacheKey() string {
	return stringx.RandUUID()
}
