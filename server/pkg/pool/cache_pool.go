package pool

import (
	"context"
	"math/rand"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"sync"
	"sync/atomic"
	"time"
)

// 缓存条目
type cacheEntry[T Conn] struct {
	conn       T            // 连接
	lastActive time.Time    // 最后活跃时间
	mu         sync.RWMutex // 保护 lastActive
}

func (e *cacheEntry[T]) getLastActive() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastActive
}

func (e *cacheEntry[T]) setLastActive(t time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastActive = t
}

type CachePool[T Conn] struct {
	factory func() (T, error)                // 创建连接的工厂方法
	cache   collx.SM[string, *cacheEntry[T]] // 使用字符串键的缓存
	config  PoolConfig[T]                    // 池配置
	closeCh chan struct{}
	closed  atomic.Bool
	mu      sync.Mutex // 保护连接创建的临界区
}

var _ Pool[Conn] = (*CachePool[Conn])(nil)

func NewCachePool[T Conn](factory func() (T, error), opts ...Option[T]) *CachePool[T] {
	config := PoolConfig[T]{
		MaxConns:            1,
		IdleTimeout:         60 * time.Minute,
		WaitTimeout:         10 * time.Second,
		HealthCheckInterval: 3 * time.Minute,
	}
	for _, opt := range opts {
		opt(&config)
	}

	p := &CachePool[T]{
		factory: factory,
		config:  config,
		closeCh: make(chan struct{}),
	}

	gox.Go(func() {
		p.backgroundMaintenance()
	})

	return p
}

// Get 获取连接（自动创建或复用缓存连接）
func (p *CachePool[T]) Get(ctx context.Context, opts ...GetOption) (T, error) {
	var zero T

	options := defaultGetOptions // 默认更新 lastActive
	for _, apply := range opts {
		apply(&options)
	}

	// 尝试查找可用连接
	if p.cache.Len() >= p.config.MaxConns {
		keys := make([]string, 0, p.cache.Len())

		p.cache.Range(func(k string, v *cacheEntry[T]) bool {
			keys = append(keys, k)
			return true
		})

		randomKey := keys[rand.Intn(len(keys))]
		entry, _ := p.cache.Load(randomKey)
		conn := entry.conn

		if options.updateLastActive {
			// 更新最后活跃时间
			entry.setLastActive(time.Now())
		}
		return conn, nil
	}

	if !options.newConn {
		return zero, ErrNoAvailableConn
	}

	// 没有找到可用连接，加锁创建
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed.Load() {
		return zero, ErrPoolClosed
	}

	// 再次检查是否已创建（防止并发）
	if p.cache.Len() >= p.config.MaxConns {
		for _, entry := range p.cache.Values() {
			if options.updateLastActive {
				entry.setLastActive(time.Now())
			}
			return entry.conn, nil
		}
	}

	// 创建新连接
	conn, err := p.factory()
	if err != nil {
		return zero, err
	}

	p.cache.Store(generateCacheKey(), &cacheEntry[T]{
		conn:       conn,
		lastActive: time.Now(),
	})

	return conn, nil
}

// Put 将连接放回缓存
func (p *CachePool[T]) Put(conn T) error {
	logx.Warn("cache pool no impl Put()")
	return nil
}

// 移除最久未使用的连接
func (p *CachePool[T]) removeOldest() {
	var oldestKey string
	var oldestTime time.Time
	p.cache.Range(func(key string, entry *cacheEntry[T]) bool {
		if oldestKey == "" || entry.getLastActive().Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.getLastActive()
		}
		return true
	})
	if oldestKey != "" {
		load, _ := p.cache.Load(oldestKey)
		p.closeConn(oldestKey, load, true)
	}
}

// Close 关闭连接池
func (p *CachePool[T]) Close() {

	if p.closed.Load() {
		return
	}

	p.closed.Store(true)
	close(p.closeCh)

	p.cache.Range(func(key string, entry *cacheEntry[T]) bool {
		// 强制关闭连接
		p.closeConn(key, entry, true)
		return true
	})
	// 触发关闭回调
	if p.config.OnPoolClose != nil {
		p.config.OnPoolClose()
	}

	p.cache.Clear()
}

// Resize 动态调整大小
func (p *CachePool[T]) Resize(newSize int) {

	if p.closed.Load() || newSize == p.config.MaxConns {
		return
	}

	p.config.MaxConns = newSize

	// 如果新大小小于当前缓存数量，清理多余的连接
	for p.cache.Len() > newSize {
		p.removeOldest()
	}
}

// Stats 获取统计信息
func (p *CachePool[T]) Stats() PoolStats {
	return PoolStats{
		TotalConns: int32(len(p.cache.Values())),
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

// cleanupIdle 清理ping失败或者闲置超时的连接
func (p *CachePool[T]) cleanupIdle() {

	p.cache.Range(func(key string, entry *cacheEntry[T]) bool {
		shouldClean := false

		// 检查是否设置了超时时间，且连接已超时
		if p.config.IdleTimeout > 0 {
			cutoff := time.Now().Add(-p.config.IdleTimeout)
			if entry.getLastActive().Before(cutoff) {
				shouldClean = true
			}
		}

		// 检查连接健康状态
		if !shouldClean && !p.ping(entry.conn) {
			shouldClean = true
		}

		if shouldClean {
			logx.Infof("cache pool - cleaning up connection (timeout or ping failed), key: %s", key)
			p.closeConn(key, entry, false)
		}
		return true
	})
}

func (p *CachePool[T]) ping(conn T) bool {
	done := make(chan struct{})
	var result bool
	gox.Go(func() {
		err := conn.Ping()
		if err != nil {
			logx.Errorf("conn pool - ping failed: %s", err.Error())
		}
		result = err == nil
		close(done)
	})
	select {
	case <-done:
		return result
	case <-time.After(5 * time.Second): // 设置超时
		logx.Debugf("cache pool - ping timeout")
		return false // 超时认为不可用
	}
}

func (p *CachePool[T]) closeConn(key string, entry *cacheEntry[T], force bool) bool {
	if !force {
		// 如果不是强制关闭且有连接关闭回调，则调用回调
		// 如果回调返回错误，则不关闭连接
		if onConnClose := p.config.OnConnClose; onConnClose != nil {
			if err := onConnClose(entry.conn); err != nil {
				logx.Infof("cache pool - connection close callback returned error, skip closing connection:: %v", err)
				return false
			}
		}
	}

	if err := entry.conn.Close(); err != nil {
		logx.Errorf("cache pool - closing connection error: %v", err)
		return false
	}
	p.cache.Delete(key)

	// 如果连接池组存在且当前缓存已空，则从组中移除该池
	if group := p.config.Group; group != nil && p.config.GroupKey != "" && p.cache.Len() == 0 {
		logx.Infof("cache pool - closing group pool, key: %s", p.config.GroupKey)
		group.Close(p.config.GroupKey)
	}

	return true
}

// 生成缓存键
func generateCacheKey() string {
	return stringx.RandUUID()
}
