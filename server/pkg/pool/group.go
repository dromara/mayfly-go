package pool

import (
	"fmt"
	"mayfly-go/pkg/logx"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type PoolGroup[T Conn] struct {
	mu          sync.RWMutex
	poolGroup   map[string]Pool[T]
	createGroup singleflight.Group
	closingWg   sync.WaitGroup
	closingMu   sync.Mutex
	closingCh   chan struct{} // 添加关闭通道
}

func NewPoolGroup[T Conn]() *PoolGroup[T] {
	return &PoolGroup[T]{
		poolGroup:   make(map[string]Pool[T]),
		createGroup: singleflight.Group{},
		closingCh:   make(chan struct{}),
	}
}

func (pg *PoolGroup[T]) GetOrCreate(
	key string,
	poolFactory func() Pool[T],
	opts ...Option[T],
) (Pool[T], error) {
	// 先尝试读锁获取
	pg.mu.RLock()
	if p, ok := pg.poolGroup[key]; ok {
		pg.mu.RUnlock()
		return p, nil
	}
	pg.mu.RUnlock()

	// 使用 singleflight 确保并发安全
	v, err, _ := pg.createGroup.Do(key, func() (any, error) {
		// 再次检查，避免在等待期间其他 goroutine 已创建
		pg.mu.RLock()
		if p, ok := pg.poolGroup[key]; ok {
			pg.mu.RUnlock()
			return p, nil
		}
		pg.mu.RUnlock()

		// 创建新池
		logx.Infof("pool group - create pool, key: %s", key)
		p := poolFactory()

		// 写入时加写锁
		pg.mu.Lock()
		pg.poolGroup[key] = p
		pg.mu.Unlock()

		return p, nil
	})

	if err != nil {
		return nil, err
	}

	return v.(Pool[T]), nil
}

// GetChanPool 获取或创建 ChannelPool 类型连接池
func (pg *PoolGroup[T]) GetChanPool(key string, factory func() (T, error), opts ...Option[T]) (Pool[T], error) {
	return pg.GetOrCreate(key, func() Pool[T] {
		return NewChannelPool(factory, opts...)
	}, opts...)
}

// GetCachePool 获取或创建 CachePool 类型连接池
func (pg *PoolGroup[T]) GetCachePool(key string, factory func() (T, error), opts ...Option[T]) (Pool[T], error) {
	return pg.GetOrCreate(key, func() Pool[T] {
		return NewCachePool(factory, opts...)
	}, opts...)
}

// Get 获取指定 key 的连接池
func (pg *PoolGroup[T]) Get(key string) (Pool[T], bool) {
	pg.mu.RLock()
	defer pg.mu.RUnlock()
	if p, ok := pg.poolGroup[key]; ok {
		return p, true
	}
	return nil, false
}

// 添加一个异步关闭的辅助函数
func (pg *PoolGroup[T]) asyncClose(pool Pool[T], key string) {
	pg.closingMu.Lock()
	pg.closingWg.Add(1)
	pg.closingMu.Unlock()

	go func() {
		defer func() {
			pg.closingMu.Lock()
			pg.closingWg.Done()
			pg.closingMu.Unlock()
		}()

		// 设置超时检测
		done := make(chan struct{})
		go func() {
			pool.Close()
			close(done)
		}()

		// 等待关闭完成或超时
		select {
		case <-done:
			logx.Infof("pool group - pool closed successfully, key: %s", key)
		case <-time.After(10 * time.Second):
			logx.Errorf("pool group - pool close timeout, key: %s", key)
			// 打印当前 goroutine 的堆栈信息
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
			logx.Errorf("pool group - goroutine stack trace:\n%s", buf)
		}
	}()
}

func (pg *PoolGroup[T]) Close(key string) error {
	pg.mu.Lock()
	if p, ok := pg.poolGroup[key]; ok {
		logx.Infof("pool group - closing pool, key: %s", key)
		pg.createGroup.Forget(key)
		delete(pg.poolGroup, key)
		pg.mu.Unlock()
		pg.asyncClose(p, key)
		return nil
	}
	pg.mu.Unlock()
	return nil
}

func (pg *PoolGroup[T]) CloseAll() {
	pg.mu.Lock()
	pools := make(map[string]Pool[T], len(pg.poolGroup))
	for k, v := range pg.poolGroup {
		pools[k] = v
	}
	pg.poolGroup = make(map[string]Pool[T])
	pg.mu.Unlock()

	// 异步关闭所有池
	for key, pool := range pools {
		pg.asyncClose(pool, key)
	}
}

// 添加一个用于监控连接池关闭状态的方法
func (pg *PoolGroup[T]) WaitForClose(timeout time.Duration) error {
	// 创建一个新的通道用于通知等待完成
	done := make(chan struct{})

	// 启动一个 goroutine 来等待所有关闭操作完成
	go func() {
		pg.closingWg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		// 在超时时打印当前状态
		pg.mu.RLock()
		remainingPools := len(pg.poolGroup)
		pg.mu.RUnlock()
		logx.Errorf("pool group - close timeout, remaining pools: %d", remainingPools)
		return fmt.Errorf("wait for pool group close timeout after %v", timeout)
	}
}

func (pg *PoolGroup[T]) AllPool() map[string]Pool[T] {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	// 返回 map 的副本，避免外部修改
	pools := make(map[string]Pool[T], len(pg.poolGroup))
	for k, v := range pg.poolGroup {
		pools[k] = v
	}
	return pools
}
