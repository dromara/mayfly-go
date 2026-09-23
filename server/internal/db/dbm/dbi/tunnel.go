package dbi

import "context"

// TunnelSpec 描述一次通道建立所需的内核侧信息。
//
// 内核只交代「经哪台中转机、访问哪个原始目标地址」，至于通道具体如何建立
// （SSH、跳板、代理等）完全由实现方决定，内核不感知其类型与细节。
type TunnelSpec struct {
	MachineId  int    // 中转机器标识，由实现方解释其含义
	RemoteAddr string // 目标原始地址，格式 host:port
}

// Tunnel 一条已建立、可被拨号的通道句柄。
//
// Host/Port 是通道建立后实际可拨的本地地址；release 是通道释放钩子，
// 由建立通道的实现方注入。句柄随连接（DbConn）一同持有，连接关闭时释放，
// 使通道生命周期显式绑定在连接上，而非依赖外部全局池按地址隐式回收。
type Tunnel struct {
	Host    string
	Port    int
	release func()
}

// NewTunnel 构造通道句柄。release 在 Close 时被调用；实现方应保证其幂等。
func NewTunnel(host string, port int, release func()) *Tunnel {
	return &Tunnel{Host: host, Port: port, release: release}
}

// Close 释放通道。nil 接收者安全（无通道的连接直接调用为无操作）。
func (t *Tunnel) Close() {
	if t != nil && t.release != nil {
		t.release()
	}
}

// TunnelOpener 通道建立能力。由具备该能力的模块在组合根注册，
// 内核仅通过此抽象在拨号前建立通道、改写目标地址，并在连接关闭时释放通道，
// 从而与具体中转实现解耦（内核不依赖任何业务模块）。
type TunnelOpener interface {
	// Open 为 spec 指向的目标建立通道，返回实际可拨地址与持有释放钩子的句柄。
	Open(ctx context.Context, spec TunnelSpec) (*Tunnel, error)
}

// tunnelOpener 全局通道建立器，启动阶段注册，运行期只读。
var tunnelOpener TunnelOpener

// RegisterTunnelOpener 注册通道建立能力实现，仅应在程序启动阶段调用。
func RegisterTunnelOpener(opener TunnelOpener) {
	if opener == nil {
		panic("dbi: register nil tunnel opener")
	}
	tunnelOpener = opener
}
