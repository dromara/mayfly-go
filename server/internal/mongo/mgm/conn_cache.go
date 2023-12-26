package mgm

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"sync"
	"time"
)

// mongo客户端连接缓存，指定时间内没有访问则会被关闭
var connCache = cache.NewTimedCache(consts.MongoConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key any, value any) {
		logx.Infof("删除mongo连接缓存: id = %v", key)
		value.(*MongoConn).Close()
	})

func init() {
	mcm.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有mongo连接实例，若存在redis实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := connCache.Items()
		for _, v := range items {
			if v.Value.(*MongoConn).Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

var mutex sync.Mutex

// 从缓存中获取mongo连接信息, 若缓存中不存在则会使用回调函数获取mongoInfo进行连接并缓存
func GetMongoConn(mongoId uint64, getMongoInfo func() (*MongoInfo, error)) (*MongoConn, error) {
	connId := getConnId(mongoId)

	// connId不为空，则为需要缓存
	needCache := connId != ""
	if needCache {
		load, ok := connCache.Get(connId)
		if ok {
			return load.(*MongoConn), nil
		}
	}

	mutex.Lock()
	defer mutex.Unlock()

	// 若缓存中不存在，则从回调函数中获取MongoInfo
	mi, err := getMongoInfo()
	if err != nil {
		return nil, err
	}

	// 连接mongo
	mc, err := mi.Conn()
	if err != nil {
		return nil, err
	}

	if needCache {
		connCache.Put(connId, mc)
	}
	return mc, nil
}

// 关闭连接，并移除缓存连接
func CloseConn(mongoId uint64) {
	connCache.Delete(mongoId)
}
