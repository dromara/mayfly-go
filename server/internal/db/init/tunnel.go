package init

import (
	"context"

	"mayfly-go/internal/db/dbm/dbi"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/mcm"
)

func init() {
	dbi.RegisterTunnelOpener(machineTunnelOpener{})
}

// machineTunnelOpener 桥接内核的通道抽象与 machine 模块的隧道实现：
// 在组合根注册，使 dbi 内核无需依赖 machine 模块即可具备经中转机建立通道的能力。
type machineTunnelOpener struct{}

func (machineTunnelOpener) Open(ctx context.Context, spec dbi.TunnelSpec) (*dbi.Tunnel, error) {
	stm, err := machineapp.GetMachineApp().GetSshTunnelMachine(ctx, spec.MachineId)
	if err != nil {
		return nil, err
	}
	able := tunnelAble{machineId: int64(spec.MachineId), remoteAddr: spec.RemoteAddr}
	host, port, err := stm.OpenSshTunnel(able)
	if err != nil {
		return nil, err
	}
	// 释放钩子绑定具体的 (中转机, 原始地址)：关闭时按此键递减通道引用计数
	return dbi.NewTunnel(host, port, func() { mcm.CloseSshTunnel(able) }), nil
}

// tunnelAble 以值形式满足隧道实现所需的地址契约（中转机器 id + 目标原始地址），
// 避免让 DbInfo 反向实现 machine 侧接口。
type tunnelAble struct {
	machineId  int64
	remoteAddr string
}

func (t tunnelAble) GetSshTunnelMachineId() int64 { return t.machineId }
func (t tunnelAble) GetRemoteAddr() string        { return t.remoteAddr }
