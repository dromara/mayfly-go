package rdm

import (
	"context"
	"mayfly-go/pkg/pool"
)

func init() {
}

var (
	poolGroup = pool.NewPoolGroup[*RedisConn]()
)

// 从缓存中获取redis连接信息, 若缓存中不存在则会使用回调函数获取redisInfo进行连接并缓存
func GetRedisConn(ctx context.Context, redisId uint64, db int, getRedisInfo func() (*RedisInfo, error)) (*RedisConn, error) {
	p, err := poolGroup.GetCachePool(getConnId(redisId, db), func() (*RedisConn, error) {
		// 若缓存中不存在，则从回调函数中获取RedisInfo
		ri, err := getRedisInfo()
		if err != nil {
			return nil, err
		}
		// 连接数据库
		return ri.Conn()
	})

	if err != nil {
		return nil, err
	}

	// 从连接池中获取一个可用的连接
	return p.Get(ctx)
}

// 移除redis连接缓存并关闭redis连接
func CloseConn(id uint64, db int) {
	poolGroup.Close(getConnId(id, db))
}
