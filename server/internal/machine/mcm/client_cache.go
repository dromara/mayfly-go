package mcm

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/logx"
	"time"
)

// 机器客户端连接缓存，指定时间内没有访问则会被关闭
var cliCache = cache.NewTimedCache(consts.MachineConnExpireTime, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(_, value any) {
		value.(*Cli).Close()
	})

func init() {
	AddCheckSshTunnelMachineUseFunc(func(machineId int) bool {
		// 遍历所有机器连接实例，若存在机器连接实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := cliCache.Items()
		for _, v := range items {
			sshTunnelMachine := v.Value.(*Cli).Info.SshTunnelMachine
			if sshTunnelMachine != nil && int(sshTunnelMachine.Id) == machineId {
				return true
			}
		}
		return false
	})
	go checkClientAvailability(3 * time.Minute)
}

// 从缓存中获取客户端信息，不存在则回调获取机器信息函数，并新建
func GetMachineCli(machineId uint64, getMachine func(uint64) (*MachineInfo, error)) (*Cli, error) {
	if load, ok := cliCache.Get(machineId); ok {
		return load.(*Cli), nil
	}

	me, err := getMachine(machineId)
	if err != nil {
		return nil, err
	}

	c, err := me.Conn()
	if err != nil {
		return nil, err
	}

	cliCache.Put(machineId, c)
	return c, nil
}

// 删除指定机器缓存客户端，并关闭客户端连接
func DeleteCli(id uint64) {
	cliCache.Delete(id)
}

// 检查缓存中的客户端是否可用，不可用则关闭客户端连接
func checkClientAvailability(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		// 遍历所有机器连接实例，若存在机器连接实例使用该ssh隧道机器，则返回true，表示还在使用中...
		items := cliCache.Items()
		for _, v := range items {
			if v == nil {
				continue
			}
			cli := v.Value.(*Cli)
			if cli.Info == nil {
				continue
			}
			if _, _, err := cli.sshClient.Conn.SendRequest("ping", true, nil); err != nil {
				logx.Errorf("machine[%s] cache client is not available: %s", cli.Info.Name, err.Error())
				DeleteCli(cli.Info.Id)
			}
			logx.Debugf("machine[%s] cache client is available", cli.Info.Name)
		}
	}
}
