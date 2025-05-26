package mcm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"
	"mayfly-go/pkg/utils/netx"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// type SshTunnelAble interface {
// 	GetSshTunnelMachineId() int
// }

var (
	// 所有检测ssh隧道机器是否被使用的函数
	checkSshTunnelMachineHasUseFuncs []CheckSshTunnelMachineHasUseFunc

	tunnelPoolGroup = pool.NewPoolGroup[*SshTunnelMachine]()
)

// 检查ssh隧道机器是否有被使用
type CheckSshTunnelMachineHasUseFunc func(int) bool

// 添加ssh隧道机器检测是否使用函数
func AddCheckSshTunnelMachineUseFunc(checkFunc CheckSshTunnelMachineHasUseFunc) {
	if checkSshTunnelMachineHasUseFuncs == nil {
		checkSshTunnelMachineHasUseFuncs = make([]CheckSshTunnelMachineHasUseFunc, 0)
	}
	checkSshTunnelMachineHasUseFuncs = append(checkSshTunnelMachineHasUseFuncs, checkFunc)
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
	_, _, err := stm.SshClient.Conn.SendRequest("ping", true, nil)
	return err
}

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

func (stm *SshTunnelMachine) OpenSshTunnel(id string, ip string, port int) (exposedIp string, exposedPort int, err error) {
	stm.mutex.Lock()
	defer stm.mutex.Unlock()

	tunnel := stm.tunnels[id]
	// 已存在该id隧道，则直接返回
	if tunnel != nil {
		// FIXME 后期改成池化连接，定时60秒检查连接可用性
		return tunnel.localHost, tunnel.localPort, nil
	}

	localPort, err := netx.GetAvailablePort()
	if err != nil {
		return "", 0, err
	}

	localHost := "127.0.0.1"
	localAddr := fmt.Sprintf("%s:%d", localHost, localPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return "", 0, err
	}

	tunnel = &Tunnel{
		id:         id,
		machineId:  stm.machineId,
		localHost:  localHost,
		localPort:  localPort,
		remoteHost: ip,
		remotePort: port,
		listener:   listener,
	}
	go tunnel.Open(stm.SshClient)
	stm.tunnels[tunnel.id] = tunnel

	return localHost, localPort, nil
}

func (stm *SshTunnelMachine) GetDialConn(network string, addr string) (net.Conn, error) {
	stm.mutex.Lock()
	defer stm.mutex.Unlock()
	return stm.SshClient.Dial(network, addr)
}

// 获取ssh隧道机器，方便统一管理充当ssh隧道的机器，避免创建多个ssh client
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
		logx.Infof("connect to the ssh tunnel machine for the first time[%d][%s:%d]", machineId, mi.Ip, mi.Port)

		return stm, err
	}, pool.WithIdleTimeout[*SshTunnelMachine](50*time.Minute), pool.WithOnConnClose(func(conn *SshTunnelMachine) error {
		mid := int(conn.mi.Id)
		logx.Debugf("periodically check if the ssh tunnel machine [%d] is still in use...", mid)

		for _, checkUseFunc := range checkSshTunnelMachineHasUseFuncs {
			// 如果一个在使用则返回不关闭，不继续后续检查
			if checkUseFunc(mid) {
				return fmt.Errorf("ssh tunnel machine [%s] is still in use", conn.mi.Name)
			}
		}

		return nil
	}))

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return pool.Get(ctx)
}

// 关闭ssh隧道机器的指定隧道
func CloseSshTunnelMachine(machineId uint64, tunnelId string) {
	sshTunnelMachinePool, ok := tunnelPoolGroup.Get(fmt.Sprintf("machine-tunnel-%d", machineId))
	if !ok {
		return
	}
	sshTunnelMachine, err := sshTunnelMachinePool.Get(context.Background())
	if err != nil {
		return
	}
	t := sshTunnelMachine.tunnels[tunnelId]
	if t != nil {
		t.Close()
		delete(sshTunnelMachine.tunnels, tunnelId)
	}
}

type Tunnel struct {
	id                string // 唯一标识
	machineId         int    // 隧道机器id
	localHost         string // 本地监听地址
	localPort         int    // 本地端口
	remoteHost        string // 远程连接地址
	remotePort        int    // 远程端口
	listener          net.Listener
	localConnections  []net.Conn
	remoteConnections []net.Conn
}

func (r *Tunnel) Open(sshClient *ssh.Client) {
	localAddr := fmt.Sprintf("%s:%d", r.localHost, r.localPort)

	for {
		logx.Debugf("隧道 %v 等待客户端访问 %v", r.id, localAddr)
		localConn, err := r.listener.Accept()
		if err != nil {
			logx.Debugf("隧道 %v 接受连接失败 %v, 退出循环", r.id, err.Error())
			logx.Debug("-------------------------------------------------")
			return
		}
		r.localConnections = append(r.localConnections, localConn)

		logx.Debugf("隧道 %v 新增本地连接 %v", r.id, localConn.RemoteAddr().String())
		remoteAddr := fmt.Sprintf("%s:%d", r.remoteHost, r.remotePort)
		logx.Debugf("隧道 %v 连接远程地址 %v ...", r.id, remoteAddr)
		remoteConn, err := sshClient.Dial("tcp", remoteAddr)
		if err != nil {
			logx.Debugf("隧道 %v 连接远程地址 %v, 退出循环", r.id, err.Error())
			logx.Debug("-------------------------------------------------")
			return
		}
		r.remoteConnections = append(r.remoteConnections, remoteConn)

		logx.Debugf("隧道 %v 连接远程主机成功", r.id)
		go r.copyConn(localConn, remoteConn)
		go r.copyConn(remoteConn, localConn)
		logx.Debugf("隧道 %v 开始转发数据 [%v]->[%v]", r.id, localAddr, remoteAddr)
		logx.Debug("~~~~~~~~~~~~~~~~~~~~分割线~~~~~~~~~~~~~~~~~~~~~~~~")
	}
}

func (r *Tunnel) Close() {
	for i := range r.localConnections {
		_ = r.localConnections[i].Close()
	}
	r.localConnections = nil
	for i := range r.remoteConnections {
		_ = r.remoteConnections[i].Close()
	}
	r.remoteConnections = nil
	_ = r.listener.Close()
	logx.Debugf("隧道 %s 监听器关闭", r.id)
}

func (r *Tunnel) copyConn(writer, reader net.Conn) {
	_, _ = io.Copy(writer, reader)
}
