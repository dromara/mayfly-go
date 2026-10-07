package mcm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"
	"mayfly-go/pkg/utils/collx"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

type SshTunnelAble interface {
	// 获取ssh隧道机器id
	GetSshTunnelMachineId() int64

	// 获取ssh隧道的远程地址
	GetRemoteAddr() string
}

var (
	tunnelPoolGroup = pool.NewPoolGroup[*SshTunnelMachine]()
)

// GetSshTunnelMachine 获取ssh隧道机器，方便统一管理充当ssh隧道的机器，避免创建多个ssh client
func GetSshTunnelMachine(ctx context.Context, machineId int, getMachine func(uint64) (*MachineInfo, error)) (*SshTunnelMachine, error) {
	pool, err := tunnelPoolGroup.GetCachePool(fmt.Sprintf("machine-tunnel-%d", machineId), func() (*SshTunnelMachine, error) {
		mi, err := getMachine(uint64(machineId))
		if err != nil {
			return nil, err
		}
		if mi == nil {
			return nil, errors.New("error get machine info")
		}
		logx.Infof("ssh tunnel machine - connect to machine for the first time - [%d][%s:%d]", machineId, mi.Ip, mi.Port)

		var clients []*ssh.Client

		// 收集所有机器信息（从内到外）
		var machines []*MachineInfo
		for mi != nil {
			machines = append(machines, mi)
			mi = mi.SshTunnelMachine
		}

		// 从最外层跳板机开始，逐层向内建立连接
		var prev *ssh.Client
		for i := len(machines) - 1; i >= 0; i-- {
			client, err := machines[i].GetSshClient(prev)
			if err != nil {
				return nil, err
			}
			clients = append(clients, client)
			prev = client
		}

		stm := &SshTunnelMachine{sshClients: clients, machineId: machineId, tunnels: map[string]*Tunnel{}}

		return stm, err
	}, pool.WithIdleTimeout[*SshTunnelMachine](0), pool.WithHealthCheckInterval[*SshTunnelMachine](1*time.Minute))

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return pool.Get(ctx)
}

// CloseSshTunnel 关闭ssh隧道（隐式隧道适配：以「机器id/远程地址」为 key）
func CloseSshTunnel(sshTunnelAble SshTunnelAble) {
	machineId := sshTunnelAble.GetSshTunnelMachineId()
	remoteAddr := sshTunnelAble.GetRemoteAddr()
	if machineId <= 0 || remoteAddr == "" {
		return
	}

	closeMachineTunnel(int(machineId), buildTunnelKey(int(machineId), remoteAddr))
}

// closeMachineTunnel 关闭机器上指定 key 的隧道，与 openTunnel 的 key 契约对称。
//
// 惰性语义：隧道机器连接已被池回收时（隧道必然早已随连接关闭）直接返回，
// 不因停止动作而新建连接
func closeMachineTunnel(machineId int, key string) {
	sshTunnelMachinePool, ok := tunnelPoolGroup.Get(fmt.Sprintf("machine-tunnel-%d", machineId))
	if !ok {
		return
	}
	sshTunnelMachine, err := sshTunnelMachinePool.Get(context.Background())
	if err != nil {
		return
	}

	sshTunnelMachine.closeTunnel(key)
}

// ssh隧道机器
type SshTunnelMachine struct {
	machineId  int                // 隧道机器id
	sshClients []*ssh.Client      // ssh客户端，可能跳转多个
	tunnels    map[string]*Tunnel // 隧道id -> 隧道

	mutex sync.RWMutex
}

/******************* pool.Conn impl *******************/

func (stm *SshTunnelMachine) Ping() error {
	_, _, err := stm.GetClient().SendRequest(
		"keepalive@openssh.com",
		true,
		nil,
	)
	return err
}

// Close 关闭ssh隧道机器及其所有隧道
func (stm *SshTunnelMachine) Close() error {
	stm.mutex.Lock()
	defer stm.mutex.Unlock()

	for id, tunnel := range stm.tunnels {
		if tunnel != nil {
			tunnel.Close()
			delete(stm.tunnels, id)
		}
	}

	if len(stm.sshClients) > 0 {
		logx.Infof("ssh tunnel machine [%d] is not in use, close tunnel...", stm.machineId)
		for _, sshClient := range stm.sshClients {
			if err := sshClient.Close(); err != nil {
				logx.Errorf("ssh tunnel machine [%d] close failed: %s", stm.machineId, err.Error())
			}
		}
	}

	return nil
}

// GetClient 获取ssh客户端
func (stm *SshTunnelMachine) GetClient() *ssh.Client {
	if len(stm.sshClients) == 0 {
		return nil
	}
	return stm.sshClients[len(stm.sshClients)-1]
}

// OpenSshTunnel 打开ssh隧道，返回暴露的ip和端口。
// 隐式隧道适配：以「机器id/远程地址」为 key，本地监听 127.0.0.1
func (stm *SshTunnelMachine) OpenSshTunnel(sshTunnelAble SshTunnelAble) (exposedIp string, exposedPort int, err error) {
	remoteAddr := sshTunnelAble.GetRemoteAddr()
	tunnel, err := stm.openTunnel(buildTunnelKey(stm.machineId, remoteAddr), "127.0.0.1:0", remoteAddr)
	if err != nil {
		return "", 0, err
	}
	return tunnel.LocalHost, tunnel.LocalPort, nil
}

// openTunnel 打开（或复用）指定 key 的隧道，返回该隧道。
//
// 这是隧道管理的唯一核心入口：key 契约由调用方定义（隐式隧道用 buildTunnelKey），
// 引用计数按 key 独立维护；未来新增转发形态（动态 SOCKS、远程转发等）只需增加适配，
// 不改本方法与 tunnels 的契约。
//
//   - key    隧道唯一标识（同 key 复用同一隧道，引用计数加一）
//   - localAddr 本地监听地址 host:port，port 为 0 时自动分配空闲端口
//   - remoteAddr 经隧道转发的远程目标地址 ip:port
func (stm *SshTunnelMachine) openTunnel(key, localAddr, remoteAddr string) (*Tunnel, error) {
	stm.mutex.Lock()
	defer stm.mutex.Unlock()

	tunnel := stm.tunnels[key]
	// 已存在该隧道，则直接返回
	if tunnel != nil {
		tunnel.refCount.Add(1)
		logx.Debugf("ssh tunnel [%s] exist, refCount: %v, localConns: %d, localAddr: %s:%d", key, tunnel.refCount.Load(), tunnel.localConns.Len(), tunnel.LocalHost, tunnel.LocalPort)
		return tunnel, nil
	}

	tunnel, err := NewTunnel(key, stm.GetClient(), localAddr, remoteAddr)
	if err != nil {
		return nil, err
	}
	stm.tunnels[key] = tunnel
	return tunnel, nil
}

// closeTunnel 释放指定 key 的隧道引用，引用归零关闭后从隧道表中移除
func (stm *SshTunnelMachine) closeTunnel(key string) {
	stm.mutex.Lock()
	defer stm.mutex.Unlock()

	t := stm.tunnels[key]
	if t == nil {
		return
	}
	t.Release()
	if t.Closed.Load() {
		logx.Infof("ssh tunnel machine - delete tunnel: %s", key)
		delete(stm.tunnels, key)
	}
}

// GetDialConn 获取通过ssh隧道连接远程地址的连接
func (stm *SshTunnelMachine) GetDialConn(network string, addr string) (net.Conn, error) {
	return stm.GetClient().Dial(network, addr)
}

type Tunnel struct {
	Id string // 唯一标识

	LocalHost  string // 本地监听地址
	LocalPort  int    // 本地端口
	RemoteAddr string // 远程连接地址

	refCount atomic.Int64 // 引用计数
	Closed   atomic.Bool  // 是否已关闭

	localListener net.Listener
	localConns    collx.SM[net.Conn, any] // net.Conn -> struct{}
}

// 创建一个隧道，localAddr 为本地监听地址 host:port（port 为 0 时自动分配空闲端口）
func NewTunnel(id string, sshClient *ssh.Client, localAddr, remoteAddr string) (*Tunnel, error) {
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, err
	}
	// 从监听器反取实际监听地址：port 为 0 时此处才能拿到系统分配的端口
	listenAddr, ok := localListener.Addr().(*net.TCPAddr)
	if !ok {
		localListener.Close()
		return nil, errors.New("failed to parse the local listening address")
	}

	tunnel := &Tunnel{
		Id:            id,
		LocalHost:     listenAddr.IP.String(),
		LocalPort:     listenAddr.Port,
		RemoteAddr:    remoteAddr,
		localListener: localListener,
	}
	tunnel.refCount.Store(1)

	gox.Go(func() {
		tunnel.Start(sshClient)
	})
	gox.Go(tunnel.startJanitor)

	logx.Infof("ssh tunnel [%s] new -> localAddr: %s", tunnel.Id, listenAddr.String())
	return tunnel, nil
}

// Start 启动隧道
func (t *Tunnel) Start(sshClient *ssh.Client) {
	localAddr := fmt.Sprintf("%s:%d", t.LocalHost, t.LocalPort)
	for {
		localConn, err := t.localListener.Accept()
		if err != nil {
			if t.Closed.Load() {
				return
			}
			logx.Errorf("ssh tunnel [%s] - localListner accept error: %v", t.Id, err)
			continue
		}
		t.localConns.Store(localConn, struct{}{})
		logx.Debugf("ssh tunnel [%s] - add local conn %v", t.Id, localConn.RemoteAddr().String())

		gox.Go(func() {
			defer func() {
				localConn.Close()
				t.localConns.Delete(localConn)
				logx.Debugf("ssh tunnel [%s] - localConn close, localConns: %d", t.Id, t.localConns.Len())
			}()

			logx.Debugf("ssh tunnel [%s] - waiting for client access %v", t.Id, localAddr)
			logx.Debugf("ssh tunnel [%s] - connecting to remote address %v ...", t.Id, t.RemoteAddr)

			remote, err := sshClient.Dial("tcp", t.RemoteAddr)
			if err != nil {
				return
			}
			defer remote.Close()

			// 使用 channel 同步双向 copy
			done := make(chan struct{}, 2)

			// 本地 -> 远程
			gox.Go(func() {
				io.Copy(remote, localConn)
				done <- struct{}{}
			})

			// 远程 -> 本地
			gox.Go(func() {
				io.Copy(localConn, remote)
				done <- struct{}{}
			})

			// 等待任意一端结束
			<-done
		})
	}
}

// Release 释放隧道引用计数
func (t *Tunnel) Release() {
	t.refCount.Add(-1)
	logx.Debugf("ssh tunnel [%s] release, refCount: %v, localConns: %d", t.Id, t.refCount.Load(), t.localConns.Len())
	if t.shouldClose() {
		t.Close()
	}
}

// Close 关闭隧道
func (t *Tunnel) Close() {
	if t.Closed.Swap(true) {
		return
	}
	logx.Infof("ssh tunnel [%s] - closed", t.Id)

	_ = t.localListener.Close()
	t.localConns.Range(func(conn net.Conn, _ any) bool {
		conn.Close()
		return true
	})
}

// startJanitor 定时检查隧道是否需要关闭
func (t *Tunnel) startJanitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if t.Closed.Load() {
			return
		}

		if t.shouldClose() {
			t.Close()
			return
		}
	}
}

// shouldClose 检查是否需要关闭
func (t *Tunnel) shouldClose() bool {
	if t.refCount.Load() > 0 {
		return false
	}
	return true
}

func buildTunnelKey(machineId int, remoteAddr string) string {
	return fmt.Sprintf("%d/%s", machineId, remoteAddr)
}
