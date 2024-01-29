package mcm

import (
	"fmt"
	"io"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/netx"
	"net"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
)

var (
	sshTunnelMachines map[int]*SshTunnelMachine = make(map[int]*SshTunnelMachine)

	mutex sync.Mutex

	// 所有检测ssh隧道机器是否被使用的函数
	checkSshTunnelMachineHasUseFuncs []CheckSshTunnelMachineHasUseFunc

	// 是否开启检查ssh隧道机器是否被使用，只有使用到了隧道机器才启用
	startCheckSshTunnelHasUse bool = false
)

// 检查ssh隧道机器是否有被使用
type CheckSshTunnelMachineHasUseFunc func(int) bool

func startCheckUse() {
	logx.Info("开启定时检测ssh隧道机器是否还有被使用")
	// 每十分钟检查一次隧道机器是否还有被使用
	scheduler.AddFun("@every 10m", func() {
		if !mutex.TryLock() {
			return
		}
		defer mutex.Unlock()
		// 遍历隧道机器，都未被使用将会被关闭
		for mid, sshTunnelMachine := range sshTunnelMachines {
			logx.Debugf("开始定时检查ssh隧道机器[%d]是否还有被使用...", mid)
			hasUse := false
			for _, checkUseFunc := range checkSshTunnelMachineHasUseFuncs {
				// 如果一个在使用则返回不关闭，不继续后续检查
				if checkUseFunc(mid) {
					hasUse = true
					break
				}
			}
			if !hasUse {
				// 都未被使用，则关闭
				sshTunnelMachine.Close()
			}
		}
	})
}

// 添加ssh隧道机器检测是否使用函数
func AddCheckSshTunnelMachineUseFunc(checkFunc CheckSshTunnelMachineHasUseFunc) {
	if checkSshTunnelMachineHasUseFuncs == nil {
		checkSshTunnelMachineHasUseFuncs = make([]CheckSshTunnelMachineHasUseFunc, 0)
	}
	checkSshTunnelMachineHasUseFuncs = append(checkSshTunnelMachineHasUseFuncs, checkFunc)
}

// ssh隧道机器
type SshTunnelMachine struct {
	machineId int // 隧道机器id
	SshClient *ssh.Client
	mutex     sync.Mutex
	tunnels   map[string]*Tunnel // 隧道id -> 隧道
}

func (stm *SshTunnelMachine) OpenSshTunnel(id string, ip string, port int) (exposedIp string, exposedPort int, err error) {
	stm.mutex.Lock()
	defer stm.mutex.Unlock()

	tunnel := stm.tunnels[id]
	// 已存在该id隧道，则直接返回
	if tunnel != nil {
		return tunnel.localHost, tunnel.localPort, nil
	}

	localPort, err := netx.GetAvailablePort()
	if err != nil {
		return "", 0, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "", 0, err
	}
	// debug
	//hostname = "0.0.0.0"

	localAddr := fmt.Sprintf("%s:%d", hostname, localPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return "", 0, err
	}

	tunnel = &Tunnel{
		id:         id,
		machineId:  stm.machineId,
		localHost:  hostname,
		localPort:  localPort,
		remoteHost: ip,
		remotePort: port,
		listener:   listener,
	}
	go tunnel.Open(stm.SshClient)
	stm.tunnels[tunnel.id] = tunnel

	return tunnel.localHost, tunnel.localPort, nil
}

func (st *SshTunnelMachine) GetDialConn(network string, addr string) (net.Conn, error) {
	st.mutex.Lock()
	defer st.mutex.Unlock()
	return st.SshClient.Dial(network, addr)
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
		logx.Infof("ssh隧道机器[%d]未被使用, 关闭隧道...", stm.machineId)
		err := stm.SshClient.Close()
		if err != nil {
			logx.Errorf("关闭ssh隧道机器[%d]发生错误: %s", stm.machineId, err.Error())
		}
	}
	delete(sshTunnelMachines, stm.machineId)
}

// 获取ssh隧道机器，方便统一管理充当ssh隧道的机器，避免创建多个ssh client
func GetSshTunnelMachine(machineId int, getMachine func(uint64) (*MachineInfo, error)) (*SshTunnelMachine, error) {
	mutex.Lock()
	defer mutex.Unlock()

	sshTunnelMachine := sshTunnelMachines[machineId]
	if sshTunnelMachine != nil {
		return sshTunnelMachine, nil
	}

	me, err := getMachine(uint64(machineId))
	if err != nil {
		return nil, err
	}

	sshClient, err := GetSshClient(me, nil)
	if err != nil {
		return nil, err
	}
	sshTunnelMachine = &SshTunnelMachine{SshClient: sshClient, machineId: machineId, tunnels: map[string]*Tunnel{}}

	logx.Infof("初次连接ssh隧道机器[%d][%s:%d]", machineId, me.Ip, me.Port)
	sshTunnelMachines[machineId] = sshTunnelMachine

	// 如果实用了隧道机器且还没开始定时检查是否还被实用，则执行定时任务检测隧道是否还被使用
	if !startCheckSshTunnelHasUse {
		startCheckUse()
		startCheckSshTunnelHasUse = true
	}
	return sshTunnelMachine, nil
}

// 关闭ssh隧道机器的指定隧道
func CloseSshTunnelMachine(machineId int, tunnelId string) {
	sshTunnelMachine := sshTunnelMachines[machineId]
	if sshTunnelMachine == nil {
		return
	}

	sshTunnelMachine.mutex.Lock()
	defer sshTunnelMachine.mutex.Unlock()
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
		go copyConn(localConn, remoteConn)
		go copyConn(remoteConn, localConn)
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

func copyConn(writer, reader net.Conn) {
	_, _ = io.Copy(writer, reader)
}
