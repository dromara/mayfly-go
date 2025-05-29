package pool

import (
	"context"
	"errors"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"sync"
	"sync/atomic"
	"time"
)

// chanConn 封装连接及其元数据
type chanConn[T Conn] struct {
	conn       T
	lastActive time.Time // 最后活跃时间
	isValid    bool      // 连接是否有效
}

func (w *chanConn[T]) Ping() error {
	if !w.isValid {
		return errors.New("connection marked invalid")
	}
	return w.conn.Ping()
}

func (w *chanConn[T]) Close() error {
	w.isValid = false
	return w.conn.Close()
}

// ChanPool 连接池结构
type ChanPool[T Conn] struct {
	mu           sync.RWMutex
	factory      func() (T, error)
	idleConns    chan *chanConn[T]
	config       PoolConfig[T]
	currentConns int32
	stats        PoolStats
	closeChan    chan struct{} // 用于关闭健康检查 goroutine
	closed       bool          // 关闭状态标识
}

// PoolStats 统计信息
type PoolStats struct {
	TotalConns  int32 // 总连接数
	IdleConns   int32 // 空闲连接数
	ActiveConns int32 // 活跃连接数
	WaitCount   int64 // 等待连接次数
}

func NewChannelPool[T Conn](factory func() (T, error), opts ...Option[T]) *ChanPool[T] {
	// 1. 初始化配置（使用默认值 + Option 覆盖）
	config := PoolConfig[T]{
		MaxConns:            5,
		IdleTimeout:         60 * time.Minute,
		WaitTimeout:         10 * time.Second,
		HealthCheckInterval: 10 * time.Minute,
	}
	for _, opt := range opts {
		opt(&config)
	}

	// 2. 创建连接池
	p := &ChanPool[T]{
		factory:   factory,
		idleConns: make(chan *chanConn[T], config.MaxConns),
		config:    config,
		closeChan: make(chan struct{}),
	}

	// 3. 启动健康检查
	go p.healthCheck()
	return p
}

func (p *ChanPool[T]) Get(ctx context.Context, opts ...GetOption) (T, error) {
	connChan := make(chan T, 1)
	errChan := make(chan error, 1)

	options := defaultGetOptions // 默认更新 lastActive
	for _, apply := range opts {
		apply(&options)
	}

	go func() {
		conn, err := p.get(options)
		if err != nil {
			errChan <- err
		} else {
			connChan <- conn
		}
	}()

	var zero T
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case err := <-errChan:
		return zero, err
	case conn := <-connChan:
		// 启动监控协程
		go func() {
			<-ctx.Done()
			// 上下文被取消后，将连接放回连接池
			if err := p.Put(conn); err != nil {
				logx.Errorf("Failed to return leaked connection: %v", err)
				conn.Close()
				atomic.AddInt32(&p.currentConns, -1)
			}
		}()
		return conn, nil
	}
}

func (p *ChanPool[T]) get(opts getOptions) (T, error) {
	var zero T
	// 检查连接池是否已关闭
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return zero, ErrPoolClosed
	}
	p.mu.RUnlock()

	// 优先从 channel 获取空闲连接（无锁）
	select {
	case wrapper := <-p.idleConns:
		atomic.AddInt32(&p.stats.IdleConns, -1)
		atomic.AddInt32(&p.stats.ActiveConns, 1)
		if opts.updateLastActive {
			wrapper.lastActive = time.Now()
		}
		return wrapper.conn, nil
	default:
		if !opts.newConn {
			return zero, ErrNoAvailableConn
		}
		return p.createConn()
	}
}

func (p *ChanPool[T]) createConn() (T, error) {
	var zero T

	// 使用CAS保证原子性
	for {
		current := atomic.LoadInt32(&p.currentConns)
		if current >= int32(p.config.MaxConns) {
			if p.config.WaitTimeout > 0 {
				return p.waitForConn()
			}
			return zero, errors.New("connection pool exhausted")
		}

		if atomic.CompareAndSwapInt32(&p.currentConns, current, current+1) {
			break
		}
	}

	// 直接创建新连接
	conn, err := p.factory()
	if err != nil {
		atomic.AddInt32(&p.currentConns, -1)
		return zero, err
	}

	// 更新状态
	atomic.AddInt32(&p.stats.ActiveConns, 1)
	return conn, nil
}

// 新增等待连接方法
func (p *ChanPool[T]) waitForConn() (T, error) {
	var zero T
	timeout := time.NewTimer(p.config.WaitTimeout)
	defer timeout.Stop()

	for {
		select {
		case wrapper := <-p.idleConns:
			if wrapper.isValid && wrapper.Ping() == nil {
				atomic.AddInt32(&p.stats.IdleConns, -1)
				atomic.AddInt32(&p.stats.ActiveConns, 1)
				wrapper.lastActive = time.Now()
				return wrapper.conn, nil
			}
			wrapper.Close()
			atomic.AddInt32(&p.currentConns, -1)
		case <-timeout.C:
			atomic.AddInt64(&p.stats.WaitCount, 1)
			return zero, errors.New("connection pool wait timeout")
		default:
			// 非阻塞检查后短暂休眠避免CPU空转
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (p *ChanPool[T]) Put(conn T) error {
	if anyx.IsBlank(conn) {
		return nil
	}

	// 检查连接池是否已关闭
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return conn.Close()
	}
	p.mu.RUnlock()

	// 快速路径
	select {
	case p.idleConns <- &chanConn[T]{conn: conn, lastActive: time.Now(), isValid: true}:
		atomic.AddInt32(&p.stats.IdleConns, 1)
		atomic.AddInt32(&p.stats.ActiveConns, -1)
		return nil
	default:
	}

	// 慢速路径
	p.mu.Lock()
	defer p.mu.Unlock()

	// 再次检查是否已关闭
	if p.closed {
		return conn.Close()
	}

	// 检查是否超过最大连接数
	if atomic.LoadInt32(&p.currentConns) > int32(p.config.MaxConns) {
		conn.Close()
		atomic.AddInt32(&p.currentConns, -1)
	} else {
		// 直接放入空闲队列
		select {
		case p.idleConns <- &chanConn[T]{conn: conn, lastActive: time.Now(), isValid: true}:
		default:
			conn.Close()
			atomic.AddInt32(&p.currentConns, -1)
		}
	}
	atomic.AddInt32(&p.stats.ActiveConns, -1)

	return nil
}

func (p *ChanPool[T]) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true

	// 1. 停止健康检查
	close(p.closeChan)

	// 2. 临时转移空闲连接
	idle := make([]*chanConn[T], 0, len(p.idleConns))
	for len(p.idleConns) > 0 {
		idle = append(idle, <-p.idleConns)
	}
	close(p.idleConns) // 安全关闭通道

	p.mu.Unlock() // 提前释放锁，避免阻塞其他操作

	// 3. 关闭所有连接（无需持有锁）
	for _, wrapper := range idle {
		wrapper.Close()
	}

	// 4. 触发关闭回调
	if p.config.OnPoolClose != nil {
		p.config.OnPoolClose()
	}
}

func (p *ChanPool[T]) healthCheck() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.checkIdleConns()
		case <-p.closeChan:
			return
		}
	}
}

func (p *ChanPool[T]) checkIdleConns() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	idle := make([]*chanConn[T], 0, len(p.idleConns))
	for len(p.idleConns) > 0 {
		idle = append(idle, <-p.idleConns)
	}

	now := time.Now()
	for _, wrapper := range idle {
		if now.Sub(wrapper.lastActive) > p.config.IdleTimeout || wrapper.Ping() != nil {
			wrapper.Close()
			atomic.AddInt32(&p.currentConns, -1)
		} else {
			select {
			case p.idleConns <- wrapper:
			default:
				wrapper.Close()
				atomic.AddInt32(&p.currentConns, -1)
			}
		}
	}
}

func (p *ChanPool[T]) Resize(newMaxConns int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	oldMax := p.config.MaxConns
	p.config.MaxConns = newMaxConns

	// 缩小连接池：关闭多余的空闲连接
	if newMaxConns < oldMax {
		toClose := oldMax - newMaxConns
		closed := 0

		// 非阻塞取出待关闭的连接
		var wrappers []*chanConn[T]
		for len(p.idleConns) > 0 && closed < toClose {
			wrappers = append(wrappers, <-p.idleConns)
			closed++
		}

		// 关闭连接并更新计数
		for _, wrapper := range wrappers {
			wrapper.Close()
			atomic.AddInt32(&p.currentConns, -1)
			atomic.AddInt32(&p.stats.IdleConns, -1)
		}
	}

	// 重建空闲连接通道（无需迁移连接，因 channel 本身无状态）
	p.idleConns = make(chan *chanConn[T], newMaxConns)
}

func (p *ChanPool[T]) CheckLeaks() []T {
	p.mu.Lock()
	defer p.mu.Unlock()

	var leaks []T
	now := time.Now()

	// 检查所有空闲连接
	idle := make([]*chanConn[T], 0, len(p.idleConns))
	for len(p.idleConns) > 0 {
		idle = append(idle, <-p.idleConns)
	}

	for _, wrapper := range idle {
		// 判定泄漏条件：长期未使用且未被标记为活跃
		if now.Sub(wrapper.lastActive) > 10*p.config.IdleTimeout {
			leaks = append(leaks, wrapper.conn)
			wrapper.Close()
			atomic.AddInt32(&p.currentConns, -1)
			atomic.AddInt32(&p.stats.IdleConns, -1)
		} else {
			// 放回空闲池
			select {
			case p.idleConns <- wrapper:
			default:
				wrapper.Close()
				atomic.AddInt32(&p.currentConns, -1)
			}
		}
	}
	return leaks
}

func (p *ChanPool[T]) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return PoolStats{
		TotalConns:  atomic.LoadInt32(&p.currentConns),
		IdleConns:   int32(len(p.idleConns)), // 直接读取通道长度
		ActiveConns: atomic.LoadInt32(&p.stats.ActiveConns),
		WaitCount:   atomic.LoadInt64(&p.stats.WaitCount),
	}
}
