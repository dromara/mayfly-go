package mcm

import (
	"context"
	"fmt"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/netx"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// 机器信息
type MachineInfo struct {
	model.ExtraData

	Id       uint64 `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Protocol int    `json:"protocol"`

	Ip   string `json:"ip"` // IP地址
	Port int    `json:"-"`  // 端口号

	AuthCertName string `json:"authCertName"`
	AuthMethod   int8   `json:"-"` // 授权认证方式
	Username     string `json:"-"` // 用户名
	Password     string `json:"-"`
	Passphrase   string `json:"-"` // 私钥口令

	SshTunnelMachine *MachineInfo `json:"-"` // ssh隧道机器
	RemoteAddr       string       `json:"-"` // ssh隧道远程地址，格式 ip:port
	EnableRecorder   int8         `json:"-"` // 是否启用终端回放记录
	CodePath         []string     `json:"codePath"`

	// HostKeyCallback 主机公钥校验回调，由 application 层按信任策略（首次信任/指纹比对/失配拒绝）装配。
	// 未装配即连接属于编程错误，连接层按拒绝处理而非静默放行
	HostKeyCallback ssh.HostKeyCallback `json:"-"`
}

var _ (SshTunnelAble) = (*MachineInfo)(nil)

func (mi *MachineInfo) GetSshTunnelMachineId() int64 {
	if mi.SshTunnelMachine == nil {
		return 0
	}
	return int64(mi.SshTunnelMachine.Id)
}

func (mi *MachineInfo) GetRemoteAddr() string {
	if mi.RemoteAddr != "" {
		return mi.RemoteAddr
	}
	return fmt.Sprintf("%s:%d", mi.Ip, mi.Port)
}

func (mi *MachineInfo) UseSshTunnel() bool {
	return mi.SshTunnelMachine != nil
}

// HostKeyAddr 返回主机公钥信任键：连接改写前的原始目标地址（ip:port）。
// 不能用 ssh 回调传入的 hostname：跳板隧道会把连接地址改写为本地暴露地址；
// RemoteAddr 在改写前已被记录为原始地址，未走隧道时 Ip/Port 本身即原始值
func (mi *MachineInfo) HostKeyAddr() string {
	if mi.RemoteAddr != "" {
		return mi.RemoteAddr
	}
	return fmt.Sprintf("%s:%d", mi.Ip, mi.Port)
}

// GetSshClient 获取ssh客户端连接
func (mi *MachineInfo) GetSshClient(jumpClient *ssh.Client) (*ssh.Client, error) {
	if mi.HostKeyCallback == nil {
		// 信任策略未装配即发起连接属于装配缺陷而非可降级场景：
		// 跳过校验等于对中间人攻击不设防，宁可让这台机器连不上
		return nil, errorx.NewBizf("machine [%s] host key verify policy is not configured, connection rejected", mi.Name)
	}
	config := &ssh.ClientConfig{
		User:            mi.Username,
		HostKeyCallback: mi.HostKeyCallback,
		Timeout:         10 * time.Second,
	}
	if ciphers := mi.GetExtraString("ciphers"); ciphers != "" {
		config.Ciphers = strings.Split(ciphers, ",")
	}
	if keyExchanges := mi.GetExtraString("keyExchanges"); keyExchanges != "" {
		config.KeyExchanges = strings.Split(keyExchanges, ",")
	}

	if mi.AuthMethod == int8(tagentity.AuthCertCiphertextTypePassword) {
		config.Auth = []ssh.AuthMethod{ssh.Password(mi.Password)}
	} else if mi.AuthMethod == int8(tagentity.AuthCertCiphertextTypePrivateKey) {
		var key ssh.Signer
		var err error

		if len(mi.Passphrase) > 0 {
			key, err = ssh.ParsePrivateKeyWithPassphrase([]byte(mi.Password), []byte(mi.Passphrase))
		} else {
			key, err = ssh.ParsePrivateKey([]byte(mi.Password))
		}
		if err != nil {
			return nil, err
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	}

	addr := fmt.Sprintf("%s:%d", mi.Ip, mi.Port)
	if jumpClient != nil {
		// 连接目标服务器
		netConn, err := jumpClient.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		conn, channel, reqs, err := ssh.NewClientConn(netConn, addr, config)
		if err != nil {
			return nil, err
		}
		// 创建目标服务器的 SSH 客户端
		return ssh.NewClient(conn, channel, reqs), nil
	}

	return ssh.Dial("tcp", addr, config)
}

// 连接
func (mi *MachineInfo) Conn(ctx context.Context) (*Cli, error) {
	logx.Infof("the machine[%s] is connecting: %s:%d", mi.Name, mi.Ip, mi.Port)

	// 如果使用了ssh隧道，则修改机器ip port为暴露的ip port
	err := mi.IfUseSshTunnelChangeIpPort(ctx, false)
	if err != nil {
		return nil, errorx.NewBizf("ssh tunnel connection failed: %s", err.Error())
	}

	cli := &Cli{Info: mi}
	sshClient, err := mi.GetSshClient(nil)
	if err != nil {
		CloseSshTunnel(mi)
		return nil, err
	}
	cli.sshClient = sshClient
	return cli, nil
}

// 如果使用了ssh隧道，则修改机器ip port为暴露的ip port
func (mi *MachineInfo) IfUseSshTunnelChangeIpPort(ctx context.Context, out bool) error {
	if !mi.UseSshTunnel() {
		return nil
	}

	mi.RemoteAddr = mi.GetRemoteAddr()
	originId := mi.Id
	if originId == 0 {
		// 随机设置一个id，如果使用了隧道则用于临时保存隧道
		mi.Id = uint64(time.Now().Nanosecond())
	}

	stm := mi.SshTunnelMachine
	sshTunnelMachine, err := GetSshTunnelMachine(ctx, int(stm.Id), func(u uint64) (*MachineInfo, error) {
		return stm, nil
	})
	if err != nil {
		return err
	}
	exposeIp, exposePort, err := sshTunnelMachine.OpenSshTunnel(mi)
	if err != nil {
		return err
	}

	// 是否获取局域网的本地IP
	if out {
		exposeIp = netx.GetOutBoundIP()
	}

	// 修改机器ip地址
	mi.Ip = exposeIp
	mi.Port = exposePort
	return nil
}
