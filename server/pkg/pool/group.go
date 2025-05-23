package pool

import (
	"mayfly-go/pkg/logx"
	"sync"

	"golang.org/x/sync/singleflight"
)

type PoolGroup[T Conn] struct {
	mu          sync.RWMutex
	poolGroup   map[string]Pool[T]
	createGroup singleflight.Group
}

func NewPoolGroup[T Conn]() *PoolGroup[T] {
	return &PoolGroup[T]{
		poolGroup:   make(map[string]Pool[T]),
		createGroup: singleflight.Group{},
	}
}

func (pg *PoolGroup[T]) GetOrCreate(
	key string,
	poolFactory func() Pool[T],
	opts ...Option,
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
func (pg *PoolGroup[T]) GetChanPool(key string, factory func() (T, error), opts ...Option) (Pool[T], error) {
	return pg.GetOrCreate(key, func() Pool[T] {
		return NewChannelPool(factory, opts...)
	}, opts...)
}

// GetCachePool 获取或创建 CachePool 类型连接池
func (pg *PoolGroup[T]) GetCachePool(key string, factory func() (T, error), opts ...Option) (Pool[T], error) {
	return pg.GetOrCreate(key, func() Pool[T] {
		return NewCachePool(factory, opts...)
	}, opts...)
}

func (pg *PoolGroup[T]) Close(key string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if p, ok := pg.poolGroup[key]; ok {
		logx.Infof("pool group - close pool, key: %s", key)
		p.Close()
		pg.createGroup.Forget(key)
		delete(pg.poolGroup, key)
	}
	return nil
}

func (pg *PoolGroup[T]) CloseAll() {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	for key := range pg.poolGroup {
		pg.poolGroup[key].Close()
		pg.createGroup.Forget(key)
	}
	pg.poolGroup = make(map[string]Pool[T])
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
