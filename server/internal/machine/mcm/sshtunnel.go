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
	"mayfly-go/pkg/utils/netx"
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
		sshClient, err := GetSshClient(mi, nil)
		if err != nil {
			return nil, err
		}
		stm := &SshTunnelMachine{SshClient: sshClient, machineId: machineId, tunnels: map[string]*Tunnel{}, mi: mi}
		logx.Infof("ssh tunnel machine - connect to machine for the first time - [%d][%s:%d]", machineId, mi.Ip, mi.Port)

		return stm, err
	}, pool.WithIdleTimeout[*SshTunnelMachine](0), pool.WithHealthCheckInterval[*SshTunnelMachine](1*time.Minute))

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return pool.Get(ctx)
}

// CloseSshTunnel 关闭ssh隧道
func CloseSshTunnel(sshTunnelAble SshTunnelAble) {
	machineId := sshTunnelAble.GetSshTunnelMachineId()
	remoteAddr := sshTunnelAble.GetRemoteAddr()
	if machineId <= 0 || remoteAddr == "" {
		return
	}

	sshTunnelMachinePool, ok := tunnelPoolGroup.Get(fmt.Sprintf("machine-tunnel-%d", machineId))
	if !ok {
		return
	}
	sshTunnelMachine, err := sshTunnelMachinePool.Get(context.Background())
	if err != nil {
		return
	}

	sshTunnelMachine.mutex.Lock()
	defer sshTunnelMachine.mutex.Unlock()

	tunnelId := buildTunnelKey(int(machineId), remoteAddr)
	t := sshTunnelMachine.tunnels[tunnelId]
	if t != nil {
		t.Release()
		if t.Closed.Load() {
			logx.Infof("ssh tunnel machine - delete tunnel: %s", tunnelId)
			delete(sshTunnelMachine.tunnels, tunnelId)
		}
	}
}

// ssh隧道机器
type SshTunnelMachine struct {
	mi        *MachineInfo
	machineId int // 隧道机器id
	SshClient *ssh.Client
	mutex     sync.Mutex
	tunnels   map[string]*Tunnel // 隧道id -> 隧道
}

/******************* pool.Conn impl *******************/

func (stm *SshTunnelMachine) Ping() error {
	_, _, err := stm.SshClient.SendRequest(
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

	if stm.SshClient != nil {
		logx.Infof("ssh tunnel machine [%d] is not in use, close tunnel...", stm.machineId)
		err := stm.SshClient.Close()
		if err != nil {
			logx.Errorf("error in closing ssh tunnel machine [%d]: %s", stm.machineId, err.Error())
		}
	}

	return nil
}

// OpenSshTunnel 打开ssh隧道，返回暴露的ip和端口
func (stm *SshTunnelMachine) OpenSshTunnel(sshTunnelAble SshTunnelAble) (exposedIp string, exposedPort int, err error) {
	stm.mutex.Lock()
	defer stm.mutex.Unlock()

	remoteAddr := sshTunnelAble.GetRemoteAddr()
	tunnelKey := buildTunnelKey(stm.machineId, remoteAddr)
	tunnel := stm.tunnels[tunnelKey]
	// 已存在该隧道，则直接返回
	if tunnel != nil {
		tunnel.refCount.Add(1)
		logx.Debugf("ssh tunnel [%s] exist, refCount: %v, localConns: %d, localAddr: %s:%d", tunnelKey, tunnel.refCount.Load(), tunnel.localConns.Len(), tunnel.LocalHost, tunnel.LocalPort)
		return tunnel.LocalHost, tunnel.LocalPort, nil
	}

	tunnel, err = NewTunnel(tunnelKey, stm.SshClient, remoteAddr)
	if err != nil {
		return "", 0, err
	}
	stm.tunnels[tunnelKey] = tunnel
	return tunnel.LocalHost, tunnel.LocalPort, nil
}

// GetDialConn 获取通过ssh隧道连接远程地址的连接
func (stm *SshTunnelMachine) GetDialConn(network string, addr string) (net.Conn, error) {
	return stm.SshClient.Dial(network, addr)
}

type Tunnel struct {
	Id        string // 唯一标识

	LocalHost  string // 本地监听地址
	LocalPort  int    // 本地端口
	RemoteAddr string // 远程连接地址

	refCount atomic.Int64 // 引用计数
	Closed   atomic.Bool // 是否已关闭

	localListener net.Listener
	localConns    collx.SM[net.Conn, any] // net.Conn -> struct{}
}

// 创建一个隧道
func NewTunnel(id string, sshClient *ssh.Client, remoteAddr string) (*Tunnel, error) {
	localPort, err := netx.GetAvailablePort()
	if err != nil {
		return nil, err
	}

	localHost := "127.0.0.1"
	localAddr := fmt.Sprintf("%s:%d", localHost, localPort)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, err
	}

	tunnel := &Tunnel{
		Id:            id,
		LocalHost:     localHost,
		LocalPort:     localPort,
		RemoteAddr:    remoteAddr,
		localListener: localListener,
	}
	tunnel.refCount.Store(1)

	gox.Go(func() {
		tunnel.Start(sshClient)
	})
	gox.Go(tunnel.startJanitor)

	logx.Infof("ssh tunnel [%s] new -> localAddr: %s", tunnel.Id, localAddr)
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
			go func() {
				io.Copy(remote, localConn)
				done <- struct{}{}
			}()

			// 远程 -> 本地
			go func() {
				io.Copy(localConn, remote)
				done <- struct{}{}
			}()

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
