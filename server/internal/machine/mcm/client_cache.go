package mcm

import (
	"errors"
	"mayfly-go/internal/common/consts"
	tagentity "mayfly-go/internal/tag/domain/entity"
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

// 从缓存中获取客户端信息，不存在则回调获取机器信息函数，并新建。
// @param 机器的授权凭证名
func GetMachineCli(authCertName string, getMachine func(string) (*MachineInfo, error)) (*Cli, error) {
	if load, ok := cliCache.Get(authCertName); ok {
		return load.(*Cli), nil
	}

	mi, err := getMachine(authCertName)
	if err != nil {
		return nil, err
	}
	mi.Key = authCertName
	c, err := mi.Conn()
	if err != nil {
		return nil, err
	}

	cliCache.Put(authCertName, c)
	return c, nil
}

// 根据机器id从已连接的机器客户端中获取特权账号连接, 若不存在特权账号，则随机返回一个
func GetMachineCliById(machineId uint64) (*Cli, error) {
	// 遍历所有机器连接实例，删除指定机器id关联的连接...
	items := cliCache.Items()

	var machineCli *Cli
	for _, v := range items {
		cli := v.Value.(*Cli)
		mi := cli.Info
		if mi.Id != machineId {
			continue
		}
		machineCli = cli

		// 如果是特权账号，则跳出
		if mi.AuthCertType == tagentity.AuthCertTypePrivileged {
			break
		}
	}

	if machineCli != nil {
		return machineCli, nil
	}
	return nil, errors.New("no connection exists for this machine id")
}

// 删除指定机器缓存客户端，并关闭客户端连接
func DeleteCli(id uint64) {
	// 遍历所有机器连接实例，删除指定机器id关联的连接...
	items := cliCache.Items()
	for _, v := range items {
		mi := v.Value.(*Cli).Info
		if mi.Id == id {
			cliCache.Delete(mi.Key)
		}
	}
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
			if cli.sshClient == nil {
				continue
			}
			if cli.sshClient.Conn == nil {
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
