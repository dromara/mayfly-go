package mcm

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/pkg/cache"
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

// 是否存在指定id的客户端连接
func HasCli(machineId uint64) bool {
	if _, ok := cliCache.Get(machineId); ok {
		return true
	}
	return false
}

// 删除指定机器客户端，并关闭客户端连接
func DeleteCli(id uint64) {
	cliCache.Delete(id)
}
