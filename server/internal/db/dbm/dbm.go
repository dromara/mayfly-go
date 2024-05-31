package dbm

import (
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/dm"
	_ "mayfly-go/internal/db/dbm/mssql"
	_ "mayfly-go/internal/db/dbm/mysql"
	_ "mayfly-go/internal/db/dbm/oracle"
	_ "mayfly-go/internal/db/dbm/postgres"
	_ "mayfly-go/internal/db/dbm/sqlite"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"sync"
	"time"
)

// 客户端连接缓存，指定时间内没有访问则会被关闭, key为数据库连接id
var connCache = cache.NewTimedCache(consts.DbConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key any, value any) {
		logx.Info(fmt.Sprintf("删除db连接缓存 id = %s", key))
		value.(*dbi.DbConn).Close()
	})

func init() {
	mcm.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有db连接实例，若存在db实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := connCache.Items()
		for _, v := range items {
			if v.Value.(*dbi.DbConn).Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

var mutex sync.Mutex

// 从缓存中获取数据库连接信息，若缓存中不存在则会使用回调函数获取dbInfo进行连接并缓存
func GetDbConn(dbId uint64, database string, getDbInfo func() (*dbi.DbInfo, error)) (*dbi.DbConn, error) {
	connId := dbi.GetDbConnId(dbId, database)

	// connId不为空，则为需要缓存
	needCache := connId != ""
	if needCache {
		load, ok := connCache.Get(connId)
		if ok {
			return load.(*dbi.DbConn), nil
		}
	}

	mutex.Lock()
	defer mutex.Unlock()

	// 若缓存中不存在，则从回调函数中获取DbInfo
	dbInfo, err := getDbInfo()
	if err != nil {
		return nil, err
	}

	// 连接数据库
	dbConn, err := Conn(dbInfo)
	if err != nil {
		return nil, err
	}

	if needCache {
		connCache.Put(connId, dbConn)
	}
	return dbConn, nil
}

// 使用指定dbInfo信息进行连接
func Conn(di *dbi.DbInfo) (*dbi.DbConn, error) {
	return di.Conn(dbi.GetMeta(di.Type))
}

// 根据实例id获取连接
func GetDbConnByInstanceId(instanceId uint64) *dbi.DbConn {
	for _, connItem := range connCache.Items() {
		conn := connItem.Value.(*dbi.DbConn)
		if conn.Info.InstanceId == instanceId {
			return conn
		}
	}
	return nil
}

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	connCache.Delete(dbi.GetDbConnId(dbId, db))
}
