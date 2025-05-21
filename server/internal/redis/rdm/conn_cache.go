package rdm

import (
	"context"
	"mayfly-go/pkg/pool"
	"time"
)

func init() {
}

var connPool = make(map[string]pool.Pool)

func getPool(redisId uint64, db int, getRedisInfo func() (*RedisInfo, error)) (pool.Pool, error) {
	connId := getConnId(redisId, db)
	// 获取连接池，如果没有，则创建一个
	if p, ok := connPool[connId]; !ok {
		var err error
		p, err = pool.NewChannelPool(&pool.Config{
			InitialCap:  1,                //资源池初始连接数
			MaxCap:      10,               //最大空闲连接数
			MaxIdle:     10,               //最大并发连接数
			IdleTimeout: 10 * time.Minute, // 连接最大空闲时间，过期则失效
			Factory: func() (interface{}, error) {
				// 若缓存中不存在，则从回调函数中获取RedisInfo
				ri, err := getRedisInfo()
				if err != nil {
					return nil, err
				}
				// 连接数据库
				return ri.Conn()
			},
			Close: func(v interface{}) error {
				v.(*RedisConn).Close()
				return nil
			},
			Ping: func(v interface{}) error {
				_, err := v.(*RedisConn).Cli.Ping(context.Background()).Result()
				return err
			},
		})
		if err != nil {
			return nil, err
		}
		connPool[connId] = p
		return p, nil
	} else {
		return p, nil
	}
}

func PutRedisConn(c *RedisConn) {
	if nil == c {
		return
	}

	if p, ok := connPool[getConnId(c.Info.Id, c.Info.Db)]; ok {
		p.Put(c)
	}
}

// 从缓存中获取redis连接信息, 若缓存中不存在则会使用回调函数获取redisInfo进行连接并缓存
func GetRedisConn(redisId uint64, db int, getRedisInfo func() (*RedisInfo, error)) (*RedisConn, error) {
	p, err := getPool(redisId, db, getRedisInfo)
	if err != nil {
		return nil, err
	}

	// 从连接池中获取一个可用的连接
	c, err := p.Get()
	if err != nil {
		return nil, err
	}
	// 用完后记的放回连接池
	return c.(*RedisConn), nil
}

// 移除redis连接缓存并关闭redis连接
func CloseConn(id uint64, db int) {
	delete(connPool, getConnId(id, db))
}
