package mgm

import (
	"context"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/pool"
)

var (
	poolGroup = pool.NewPoolGroup[*MongoConn]()
)

func init() {
	mcm.AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		items := poolGroup.AllPool()
		for _, v := range items {
			conn, err := v.Get(context.Background(), pool.WithGetNoUpdateLastActive(), pool.WithGetNoNewConn())
			if err != nil {
				continue // 获取连接失败，跳过
			}
			if conn.Info.SshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

// 从缓存中获取mongo连接信息, 若缓存中不存在则会使用回调函数获取mongoInfo进行连接并缓存
func GetMongoConn(ctx context.Context, mongoId uint64, getMongoInfo func() (*MongoInfo, error)) (*MongoConn, error) {
	pool, err := poolGroup.GetCachePool(getConnId(mongoId), func() (*MongoConn, error) {
		// 若缓存中不存在，则从回调函数中获取MongoInfo
		mi, err := getMongoInfo()
		if err != nil {
			return nil, err
		}

		// 连接mongo
		return mi.Conn()
	})

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return pool.Get(ctx)
}

// 关闭连接，并移除缓存连接
func CloseConn(mongoId uint64) {
	poolGroup.Close(getConnId(mongoId))
}
