package pool

import (
	"mayfly-go/pkg/logx"

	"golang.org/x/sync/singleflight"
)

type PoolGroup[T Conn] struct {
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
	if p, ok := pg.poolGroup[key]; ok {
		return p, nil
	}

	v, err, _ := pg.createGroup.Do(key, func() (any, error) {
		logx.Infof("pool group - create pool, key: %s", key)
		p := poolFactory()
		pg.poolGroup[key] = p
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
	if p, ok := pg.poolGroup[key]; ok {
		logx.Infof("pool group - close pool, key: %s", key)
		p.Close()
		pg.createGroup.Forget(key)
		delete(pg.poolGroup, key)
	}
	return nil
}

func (pg *PoolGroup[T]) CloseAll() {
	for key := range pg.poolGroup {
		pg.Close(key)
	}
}

func (pg *PoolGroup[T]) AllPool() map[string]Pool[T] {
	return pg.poolGroup
}
