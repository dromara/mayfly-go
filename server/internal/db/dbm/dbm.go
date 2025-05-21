package dbm

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/dm"
	_ "mayfly-go/internal/db/dbm/mssql"
	_ "mayfly-go/internal/db/dbm/mysql"
	_ "mayfly-go/internal/db/dbm/oracle"
	_ "mayfly-go/internal/db/dbm/postgres"
	_ "mayfly-go/internal/db/dbm/sqlite"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"
	"time"
)

var connPool = make(map[string]pool.Pool)
var instPool = make(map[uint64]pool.Pool)

func init() {
}

// PutDbConn 释放连接
func PutDbConn(c *dbi.DbConn) {
	if nil == c {
		return
	}
	connId := dbi.GetDbConnId(c.Info.Id, c.Info.Database)
	if p, ok := connPool[connId]; ok {
		p.Put(c)
	}
}

func getPool(dbId uint64, database string, getDbInfo func() (*dbi.DbInfo, error)) (pool.Pool, error) {
	connId := dbi.GetDbConnId(dbId, database)

	// 获取连接池，如果没有，则创建一个
	if p, ok := connPool[connId]; !ok {
		var err error
		p, err = pool.NewChannelPool(&pool.Config{
			InitialCap:  1,                //资源池初始连接数
			MaxCap:      10,               //最大空闲连接数
			MaxIdle:     10,               //最大并发连接数
			IdleTimeout: 10 * time.Minute, // 连接最大空闲时间，过期则失效
			Factory: func() (interface{}, error) {
				// 若缓存中不存在，则从回调函数中获取DbInfo
				dbInfo, err := getDbInfo()
				if err != nil {
					return nil, err
				}

				// 连接数据库
				return Conn(dbInfo)
			},
			Close: func(v interface{}) error {
				v.(*dbi.DbConn).Close()
				return nil
			},
			Ping: func(v interface{}) error {
				return v.(*dbi.DbConn).Ping()
			},
		})
		if err != nil {
			return nil, err
		}
		connPool[connId] = p
		instPool[dbId] = p
		return p, nil
	} else {
		return p, nil
	}
}

// GetDbConn 从连接池中获取连接信息，记的用完连接后必须调用 PutDbConn 还回池
func GetDbConn(dbId uint64, database string, getDbInfo func() (*dbi.DbInfo, error)) (*dbi.DbConn, error) {

	p, err := getPool(dbId, database, getDbInfo)
	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	c, err := p.Get()
	if err != nil {
		return nil, err
	}
	ec := c.(*dbi.DbConn)
	return ec, nil

}

// 使用指定dbInfo信息进行连接
func Conn(di *dbi.DbInfo) (*dbi.DbConn, error) {
	return di.Conn(dbi.GetMeta(di.Type))
}

// 根据实例id获取连接
func GetDbConnByInstanceId(instanceId uint64) *dbi.DbConn {
	if p, ok := instPool[instanceId]; ok {
		c, err := p.Get()
		if err != nil {
			logx.Error(fmt.Sprintf("实例id[%d]连接获取失败：%s", instanceId, err))
			return nil
		}
		return c.(*dbi.DbConn)
	}
	return nil
}

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	delete(connPool, dbi.GetDbConnId(dbId, db))
	delete(instPool, dbId)
}
