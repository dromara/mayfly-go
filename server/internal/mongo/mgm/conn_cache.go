package mgm

import (
	"context"
	"mayfly-go/pkg/pool"
	"time"
)

var connPool = make(map[string]pool.Pool)

func init() {
}

func getPool(mongoId uint64, getMongoInfo func() (*MongoInfo, error)) (pool.Pool, error) {
	connId := getConnId(mongoId)

	// 获取连接池，如果没有，则创建一个
	if p, ok := connPool[connId]; !ok {
		var err error
		p, err = pool.NewChannelPool(&pool.Config{
			InitialCap:  1,                //资源池初始连接数
			MaxCap:      10,               //最大空闲连接数
			MaxIdle:     10,               //最大并发连接数
			IdleTimeout: 10 * time.Minute, // 连接最大空闲时间，过期则失效
			Factory: func() (interface{}, error) {
				// 若缓存中不存在，则从回调函数中获取MongoInfo
				mi, err := getMongoInfo()
				if err != nil {
					return nil, err
				}

				// 连接mongo
				return mi.Conn()
			},
			Close: func(v interface{}) error {
				v.(*MongoConn).Close()
				return nil
			},
			Ping: func(v interface{}) error {
				return v.(*MongoConn).Cli.Ping(context.Background(), nil)
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

func PutMongoConn(c *MongoConn) {
	if nil == c {
		return
	}
	if p, ok := connPool[getConnId(c.Info.Id)]; ok {
		p.Put(c)
	}
}

// 从缓存中获取mongo连接信息, 若缓存中不存在则会使用回调函数获取mongoInfo进行连接并缓存
func GetMongoConn(mongoId uint64, getMongoInfo func() (*MongoInfo, error)) (*MongoConn, error) {
	p, err := getPool(mongoId, getMongoInfo)
	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	c, err := p.Get()
	if err != nil {
		return nil, err
	}
	return c.(*MongoConn), nil
}

// 关闭连接，并移除缓存连接
func CloseConn(mongoId uint64) {
	connId := getConnId(mongoId)
	delete(connPool, connId)
}
