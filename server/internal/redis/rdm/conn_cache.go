package rdm

import (
	"context"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/pool"
)

var (
	poolGroup = pool.NewPoolGroup[*RedisConn]()
)

func init() {
	mcm.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有redis连接实例，若存在redis实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := poolGroup.AllPool()
		for _, v := range items {
			rc, err := v.Get(context.Background(), pool.WithGetNoUpdateLastActive(), pool.WithGetNoNewConn())
			if err != nil {
				continue // 获取连接失败，跳过
			}
			if rc.Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

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
