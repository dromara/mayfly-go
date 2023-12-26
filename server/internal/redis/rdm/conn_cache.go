package rdm

import (
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"sync"
	"time"
)

// redis客户端连接缓存，指定时间内没有访问则会被关闭
var connCache = cache.NewTimedCache(consts.RedisConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key any, value any) {
		logx.Info(fmt.Sprintf("删除redis连接缓存 id = %s", key))
		value.(*RedisConn).Close()
	})

func init() {
	mcm.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有redis连接实例，若存在redis实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := connCache.Items()
		for _, v := range items {
			if v.Value.(*RedisConn).Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

var mutex sync.Mutex

// 从缓存中获取redis连接信息, 若缓存中不存在则会使用回调函数获取redisInfo进行连接并缓存
func GetRedisConn(redisId uint64, db int, getRedisInfo func() (*RedisInfo, error)) (*RedisConn, error) {
	connId := getConnId(redisId, db)

	// connId不为空，则为需要缓存
	needCache := connId != ""
	if needCache {
		load, ok := connCache.Get(connId)
		if ok {
			return load.(*RedisConn), nil
		}
	}

	mutex.Lock()
	defer mutex.Unlock()

	// 若缓存中不存在，则从回调函数中获取RedisInfo
	ri, err := getRedisInfo()
	if err != nil {
		return nil, err
	}

	// 连接数据库
	rc, err := ri.Conn()
	if err != nil {
		return nil, err
	}

	if needCache {
		connCache.Put(connId, rc)
	}
	return rc, nil
}

// 移除redis连接缓存并关闭redis连接
func CloseConn(id uint64, db int) {
	connCache.Delete(getConnId(id, db))
}
