package mcm

import (
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

var (
	mutex sync.Mutex

	// 所有检测ssh隧道机器是否被使用的函数
	checkSshTunnelMachineHasUseFuncs []CheckSshTunnelMachineHasUseFunc

	tunnelPool = make(map[int]pool.Pool)
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

func (stm *SshTunnelMachine) Ping() error {
	_, _, err := stm.SshClient.Conn.SendRequest("ping", true, nil)
	return err
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

func (stm *SshTunnelMachine) Close() {
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
	delete(tunnelPool, stm.machineId)
}

func getTunnelPool(machineId int, getMachine func(uint64) (*MachineInfo, error)) (pool.Pool, error) {
	// 获取连接池，如果没有，则创建一个
	if p, ok := tunnelPool[machineId]; !ok {
		var err error
		p, err = pool.NewChannelPool(&pool.Config{
			InitialCap:  1,                //资源池初始连接数
			MaxCap:      10,               //最大空闲连接数
			MaxIdle:     10,               //最大并发连接数
			IdleTimeout: 10 * time.Minute, // 连接最大空闲时间，过期则失效
			Factory: func() (interface{}, error) {
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
			},
			Close: func(v interface{}) error {
				v.(*SshTunnelMachine).Close()
				return nil
			},
			Ping: func(v interface{}) error {
				return v.(*SshTunnelMachine).Ping()
			},
		})
		if err != nil {
			return nil, err
		}
		tunnelPool[machineId] = p
		return p, nil
	} else {
		return p, nil
	}
}

// 获取ssh隧道机器，方便统一管理充当ssh隧道的机器，避免创建多个ssh client
func GetSshTunnelMachine(machineId int, getMachine func(uint64) (*MachineInfo, error)) (*SshTunnelMachine, error) {
	p, err := getTunnelPool(machineId, getMachine)
	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	c, err := p.Get()
	if err != nil {
		return nil, err
	}

	return c.(*SshTunnelMachine), nil
}

// 关闭ssh隧道机器的指定隧道
func CloseSshTunnelMachine(machineId uint64, tunnelId string) {
	//sshTunnelMachine := mcIdPool[machineId]
	//if sshTunnelMachine == nil {
	//	return
	//}
	//
	//sshTunnelMachine.mutex.Lock()
	//defer sshTunnelMachine.mutex.Unlock()
	//t := sshTunnelMachine.tunnels[tunnelId]
	//if t != nil {
	//	t.Close()
	//	delete(sshTunnelMachine.tunnels, tunnelId)
	//}
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
