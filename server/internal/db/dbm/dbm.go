package dbm

import (
	"context"
	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/dm"
	_ "mayfly-go/internal/db/dbm/mssql"
	_ "mayfly-go/internal/db/dbm/mysql"
	_ "mayfly-go/internal/db/dbm/oracle"
	_ "mayfly-go/internal/db/dbm/postgres"
	_ "mayfly-go/internal/db/dbm/sqlite"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"
)

var (
	poolGroup = pool.NewPoolGroup[*dbi.DbConn]()
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

// GetDbConn 从连接池中获取连接信息
func GetDbConn(ctx context.Context, dbId uint64, database string, getDbInfo func() (*dbi.DbInfo, error)) (*dbi.DbConn, error) {
	connId := dbi.GetDbConnId(dbId, database)

	pool, err := poolGroup.GetCachePool(connId, func() (*dbi.DbConn, error) {
		// 若缓存中不存在，则从回调函数中获取DbInfo
		dbInfo, err := getDbInfo()
		if err != nil {
			return nil, err
		}
		logx.Debugf("dbm - conn create, connId: %s, dbInfo: %v", connId, dbInfo)
		// 连接数据库
		return Conn(context.Background(), dbInfo)
	})

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return pool.Get(ctx)
}

// 使用指定dbInfo信息进行连接
func Conn(ctx context.Context, di *dbi.DbInfo) (*dbi.DbConn, error) {
	return di.Conn(ctx, dbi.GetMeta(di.Type))
}

// 根据实例id获取连接
func GetDbConnByInstanceId(ctx context.Context, instanceId uint64) *dbi.DbConn {
	for _, pool := range poolGroup.AllPool() {
		conn, err := pool.Get(ctx)
		if err != nil {
			continue
		}
		if conn.Info.InstanceId == instanceId {
			return conn
		}
	}
	return nil
}

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	poolGroup.Close(dbi.GetDbConnId(dbId, db))
}
