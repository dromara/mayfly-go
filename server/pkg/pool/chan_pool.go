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

	// 「谁持有这条连接」的决策必须在写锁内做：Close 在持锁期间置 closed 并排空空闲队列，
	// 而「先无锁查 closed、再发送」之间存在窗口——并发时连接会被塞进已经废弃的队列
	// 永久泄漏，若队列已被关闭则直接 send-on-closed panic
	discard, poolClosed := p.recycle(conn)
	if !discard {
		return nil
	}

	// Close 是 IO（要走协议报文、等对端响应），必须放到锁外做：一条卡住的连接不该把
	// 整池的 Get/Put 都挡在写锁上，而那正是慢连接最容易出现的时候
	if poolClosed {
		// 池已关闭，调用方是最后一个碰这条连接的人，把关闭结果如实交回去
		return conn.Close()
	}
	// 池还活着时的丢弃是正常的水位控制（超出 MaxConns、队列已满），不是调用方归还失败；
	// 这里的 Close 只是清理，它的错误不该被翻译成 Put 失败让调用方去重试
	conn.Close()
	return nil
}

// recycle 在写锁内裁定一条归还连接的归宿：是否要丢弃（discard），以及池是否已关闭（poolClosed）。
//
// 这里只做决策与计数，不做任何 IO——连接的 Close 由调用方在锁外执行。
func (p *ChanPool[T]) recycle(conn T) (discard, poolClosed bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return true, true
	}

	// 快速路径：队列有位置就直接回收
	select {
	case p.idleConns <- &chanConn[T]{conn: conn, lastActive: time.Now(), isValid: true}:
		atomic.AddInt32(&p.stats.IdleConns, 1)
		atomic.AddInt32(&p.stats.ActiveConns, -1)
		return false, false
	default:
	}

	// 队列已满：超出水位就直接丢弃，否则再试一次入队
	if atomic.LoadInt32(&p.currentConns) > int32(p.config.MaxConns) {
		atomic.AddInt32(&p.currentConns, -1)
	} else {
		select {
		case p.idleConns <- &chanConn[T]{conn: conn, lastActive: time.Now(), isValid: true}:
			atomic.AddInt32(&p.stats.ActiveConns, -1)
			return false, false
		default:
			atomic.AddInt32(&p.currentConns, -1)
		}
	}
	atomic.AddInt32(&p.stats.ActiveConns, -1)

	return true, false
}

// drainIdle 在写锁内取走当前全部空闲连接，取走后这些连接归调用方独占。
//
// 存在的理由：Ping 与 Close 都是网络 IO，绝不能持锁做——一轮健康检查要 Ping 全部空闲连接，
// 一个慢连接就能把整池的 Get/Put 挡在写锁上。因此统一改成「锁内排空 → 锁外体检 → 锁内回填」
func (p *ChanPool[T]) drainIdle() []*chanConn[T] {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 池已关闭时空闲连接由 Close 负责回收，这里不能再抢
	if p.closed {
		return nil
	}

	idle := make([]*chanConn[T], 0, len(p.idleConns))
	for len(p.idleConns) > 0 {
		idle = append(idle, <-p.idleConns)
	}
	return idle
}

// refillIdle 在写锁内把仍可用的连接放回空闲队列，返回必须丢弃的连接。
// 丢弃动作（Close 是 IO）留给调用方在锁外执行
func (p *ChanPool[T]) refillIdle(wrappers []*chanConn[T]) (discard []*chanConn[T]) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, wrapper := range wrappers {
		// 排空之后池被关闭了：整池计数已无意义，连接交给调用方关掉即可
		if p.closed {
			discard = append(discard, wrapper)
			continue
		}
		select {
		case p.idleConns <- wrapper:
		default:
			discard = append(discard, wrapper)
			atomic.AddInt32(&p.currentConns, -1)
		}
	}
	return discard
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
	//
	// 这里不 close(p.idleConns)：没有任何消费端 range 该通道（healthCheck 靠 closeChan 退出），
	// 关闭它换不来任何语义，只会让并发中的 Put 撞上 send-on-closed；连接回收由上面的排空 +
	// 下面的逐个 Close 完成
	idle := make([]*chanConn[T], 0, len(p.idleConns))
	for len(p.idleConns) > 0 {
		idle = append(idle, <-p.idleConns)
	}

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
	drained := p.drainIdle()
	if len(drained) == 0 {
		return
	}

	now := time.Now()
	keep := make([]*chanConn[T], 0, len(drained))
	for _, wrapper := range drained {
		// Ping 是网络往返，此处已不持锁
		if now.Sub(wrapper.lastActive) > p.config.IdleTimeout || wrapper.Ping() != nil {
			wrapper.Close()
			atomic.AddInt32(&p.currentConns, -1)
			continue
		}
		keep = append(keep, wrapper)
	}

	for _, wrapper := range p.refillIdle(keep) {
		wrapper.Close()
	}
}

func (p *ChanPool[T]) Resize(newMaxConns int) {
	// 换通道前必须先把里面的活连接取出来：直接丢掉旧通道会让它们既回不了池、也永远不会被 Close
	drained := p.drainIdle()

	p.mu.Lock()
	p.config.MaxConns = newMaxConns
	// 容量必须跟着变，否则扩容后 Put 永远填不满、缩容后也拦不住
	p.idleConns = make(chan *chanConn[T], newMaxConns)
	keep := drained
	if len(keep) > newMaxConns {
		keep = keep[:newMaxConns]
	}
	// 容量已按 keep 的长度留足，这里的发送不会阻塞，因此可以持锁做
	for _, wrapper := range keep {
		p.idleConns <- wrapper
	}
	p.mu.Unlock()

	// 超出新水位的部分关掉，Close 是 IO，放到锁外
	for _, wrapper := range drained[len(keep):] {
		wrapper.Close()
		atomic.AddInt32(&p.currentConns, -1)
		atomic.AddInt32(&p.stats.IdleConns, -1)
	}
}

func (p *ChanPool[T]) CheckLeaks() []T {
	drained := p.drainIdle()

	now := time.Now()
	var leaks []T
	keep := make([]*chanConn[T], 0, len(drained))
	for _, wrapper := range drained {
		// 判定泄漏条件：长期未使用
		if now.Sub(wrapper.lastActive) > 10*p.config.IdleTimeout {
			leaks = append(leaks, wrapper.conn)
			wrapper.Close()
			atomic.AddInt32(&p.currentConns, -1)
			atomic.AddInt32(&p.stats.IdleConns, -1)
			continue
		}
		keep = append(keep, wrapper)
	}

	for _, wrapper := range p.refillIdle(keep) {
		wrapper.Close()
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
