package dbm

import (
	"fmt"
	"mayfly-go/internal/common/consts"
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
		value.(*DbConn).Close()
	})

func init() {
	mcm.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有db连接实例，若存在db实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := connCache.Items()
		for _, v := range items {
			if v.Value.(*DbConn).Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

var mutex sync.Mutex

// 从缓存中获取数据库连接信息，若缓存中不存在则会使用回调函数获取dbInfo进行连接并缓存
func GetDbConn(dbId uint64, database string, getDbInfo func() (*DbInfo, error)) (*DbConn, error) {
	connId := GetDbConnId(dbId, database)

	// connId不为空，则为需要缓存
	needCache := connId != ""
	if needCache {
		load, ok := connCache.Get(connId)
		if ok {
			return load.(*DbConn), nil
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
	dbConn, err := dbInfo.Conn()
	if err != nil {
		return nil, err
	}

	if needCache {
		connCache.Put(connId, dbConn)
	}
	return dbConn, nil
}

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	connCache.Delete(GetDbConnId(dbId, db))
}
