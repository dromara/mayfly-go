package mcm

import (
	"context"
	"fmt"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/netx"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// 机器信息
type MachineInfo struct {
	model.ExtraData

	Key      string `json:"key"` // 缓存key
	Id       uint64 `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Protocol int    `json:"protocol"`

	Ip   string `json:"ip"` // IP地址
	Port int    `json:"-"`  // 端口号

	AuthCertName string                 `json:"authCertName"`
	AuthCertType tagentity.AuthCertType `json:"-"`
	AuthMethod   int8                   `json:"-"` // 授权认证方式
	Username     string                 `json:"-"` // 用户名
	Password     string                 `json:"-"`
	Passphrase   string                 `json:"-"` // 私钥口令

	SshTunnelMachine *MachineInfo `json:"-"` // ssh隧道机器
	TempSshMachineId uint64       `json:"-"` // ssh隧道机器id，用于记录隧道机器id，连接出错后关闭隧道
	EnableRecorder   int8         `json:"-"` // 是否启用终端回放记录
	CodePath         []string     `json:"codePath"`
}

func (mi *MachineInfo) UseSshTunnel() bool {
	return mi.SshTunnelMachine != nil
}

func (mi *MachineInfo) GetTunnelId() string {
	return fmt.Sprintf("machine:%d", mi.Id)
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
	sshClient, err := GetSshClient(mi, nil)
	if err != nil {
		if mi.UseSshTunnel() {
			CloseSshTunnelMachine(mi.TempSshMachineId, mi.GetTunnelId())
		}
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
	exposeIp, exposePort, err := sshTunnelMachine.OpenSshTunnel(mi.GetTunnelId(), mi.Ip, mi.Port)
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
	// 代理之后置空跳板机信息，防止重复跳
	mi.TempSshMachineId = stm.Id
	mi.SshTunnelMachine = nil
	return nil
}

func GetSshClient(m *MachineInfo, jumpClient *ssh.Client) (*ssh.Client, error) {
	// 递归一直取到底层没有跳板机的机器信息
	if m.SshTunnelMachine != nil {
		jumpClient, err := GetSshClient(m.SshTunnelMachine, jumpClient)
		if err != nil {
			return nil, err
		}
		// 新建一个没有跳板机的机器信息
		m1 := &MachineInfo{
			Ip:         m.Ip,
			Port:       m.Port,
			AuthMethod: m.AuthMethod,
			Username:   m.Username,
			Password:   m.Password,
			Passphrase: m.Passphrase,
		}
		// 使用跳板机连接目标机器
		return GetSshClient(m1, jumpClient)
	}
	// 配置 SSH 客户端
	config := &ssh.ClientConfig{
		User: m.Username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 5 * time.Second,
	}
	if ciphers := m.GetExtraString("ciphers"); ciphers != "" {
		config.Ciphers = strings.Split(ciphers, ",")
	}
	if keyExchanges := m.GetExtraString("keyExchanges"); keyExchanges != "" {
		config.KeyExchanges = strings.Split(keyExchanges, ",")
	}

	if m.AuthMethod == int8(tagentity.AuthCertCiphertextTypePassword) {
		config.Auth = []ssh.AuthMethod{ssh.Password(m.Password)}
	} else if m.AuthMethod == int8(tagentity.AuthCertCiphertextTypePrivateKey) {
		var key ssh.Signer
		var err error

		if len(m.Passphrase) > 0 {
			key, err = ssh.ParsePrivateKeyWithPassphrase([]byte(m.Password), []byte(m.Passphrase))
		} else {
			key, err = ssh.ParsePrivateKey([]byte(m.Password))
		}
		if err != nil {
			return nil, err
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	}

	addr := fmt.Sprintf("%s:%d", m.Ip, m.Port)
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
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return sshClient, nil
}
