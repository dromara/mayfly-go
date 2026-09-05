package pool

import (
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type PoolGroup[T Conn] struct {
	poolGroup   collx.SM[string, Pool[T]]
	createGroup singleflight.Group
	closingWg   sync.WaitGroup
	closingMu   sync.Mutex
	closingCh   chan struct{} // 添加关闭通道
}

func NewPoolGroup[T Conn]() *PoolGroup[T] {
	return &PoolGroup[T]{
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
	if p, ok := pg.poolGroup.Load(key); ok {
		return p, nil
	}
	// 使用 singleflight 确保并发安全
	v, err, _ := pg.createGroup.Do(key, func() (any, error) {
		// 再次检查，避免在等待期间其他 goroutine 已创建
		if p, ok := pg.poolGroup.Load(key); ok {
			return p, nil
		}

		// 创建新池
		logx.Infof("pool group - create pool, key: %s", key)
		p := poolFactory()

		// 写入时加写锁
		pg.poolGroup.Store(key, p)

		return p, nil
	})

	if err != nil {
		return nil, err
	}

	return v.(Pool[T]), nil
}

// GetChanPool 获取或创建 ChannelPool 类型连接池
// key: 连接池标识
// factory: 连接创建函数
// opts: 配置项
func (pg *PoolGroup[T]) GetChanPool(key string, factory func() (T, error), opts ...Option[T]) (Pool[T], error) {
	return pg.GetOrCreate(key, func() Pool[T] {
		opts = append(opts, WithGroup(pg), WithGroupKey[T](key))
		return NewChannelPool(factory, opts...)
	}, opts...)
}

// GetCachePool 获取或创建 CachePool 类型连接池
// key: 连接池标识
// factory: 连接创建函数
// opts: 配置项
func (pg *PoolGroup[T]) GetCachePool(key string, factory func() (T, error), opts ...Option[T]) (Pool[T], error) {
	return pg.GetOrCreate(key, func() Pool[T] {
		opts = append(opts, WithGroup(pg), WithGroupKey[T](key))
		return NewCachePool(factory, opts...)
	}, opts...)
}

// Get 获取指定 key 的连接池
func (pg *PoolGroup[T]) Get(key string) (Pool[T], bool) {
	return pg.poolGroup.Load(key)
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
	if p, ok := pg.poolGroup.Load(key); ok {
		logx.Infof("pool group - closing pool, key: %s", key)
		pg.createGroup.Forget(key)
		pg.poolGroup.Delete(key)
		pg.asyncClose(p, key)
		return nil
	}
	return nil
}

func (pg *PoolGroup[T]) CloseAll() {
	// 先收集所有 key，再逐个走 Close(key)，确保与 Close 行为一致：
	// 不仅关闭池，还要从注册表中摘除，避免关闭后 GetOrCreate 命中已关闭的死池
	keys := make([]string, 0)
	pg.poolGroup.Range(func(k string, _ Pool[T]) bool {
		keys = append(keys, k)
		return true
	})
	for _, k := range keys {
		_ = pg.Close(k)
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
		remainingPools := len(pg.poolGroup.Values())
		logx.Errorf("pool group - close timeout, remaining pools: %d", remainingPools)
		return fmt.Errorf("wait for pool group close timeout after %v", timeout)
	}
}

func (pg *PoolGroup[T]) AllPool() []Pool[T] {
	return pg.poolGroup.Values()
}
