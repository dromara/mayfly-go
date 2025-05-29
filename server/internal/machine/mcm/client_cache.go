package mcm

import (
	"context"
	"fmt"
	"mayfly-go/pkg/pool"
)

var (
	poolGroup = pool.NewPoolGroup[*Cli]()
)

func init() {
	AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有redis连接实例，若存在redis实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := poolGroup.AllPool()
		for _, v := range items {
			if v.Stats().TotalConns == 0 {
				continue // 连接池中没有连接，跳过
			}
			cli, err := v.Get(context.Background())
			if err != nil {
				continue // 获取连接失败，跳过
			}
			sshTunnelMachine := cli.Info.SshTunnelMachine
			if sshTunnelMachine != nil && sshTunnelMachine.Id == uint64(machineId) {
				return true
			}
		}
		return false
	})
}

// 从缓存中获取客户端信息，不存在则回调获取机器信息函数，并新建。
// @param 机器的授权凭证名
func GetMachineCli(ctx context.Context, authCertName string, getMachine func(string) (*MachineInfo, error)) (*Cli, error) {
	pool, err := poolGroup.GetCachePool(authCertName, func() (*Cli, error) {
		mi, err := getMachine(authCertName)
		if err != nil {
			return nil, err
		}
		mi.Key = authCertName
		return mi.Conn(context.Background())
	})

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return pool.Get(ctx)
}

// 删除指定机器缓存客户端，并关闭客户端连接
func DeleteCli(id uint64) {
	for _, p := range poolGroup.AllPool() {
		conn, err := p.Get(context.Background(), pool.WithGetNoUpdateLastActive(), pool.WithGetNoNewConn())
		if err != nil {
			continue
		}
		if conn.Info.Id == id {
			poolGroup.Close(conn.Info.AuthCertName)
		}
	}
	// 删除隧道
	tunnelPoolGroup.Close(fmt.Sprintf("machine-tunnel-%d", id))
}
