package machine

import (
	"errors"
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/global"
	"net"
	"time"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

// 机器信息
type Info struct {
	Id   uint64
	Name string `json:"name"`
	Ip   string `json:"ip"`   // IP地址
	Port int    `json:"port"` // 端口号

	AuthMethod int8   `json:"authMethod"` // 授权认证方式
	Username   string `json:"username"`   // 用户名
	Password   string `json:"-"`
	Passphrase string `json:"-"` // 私钥口令

	Status             int8   `json:"status"`             // 状态 1:启用；2:停用
	SshTunnelMachineId int    `json:"sshTunnelMachineId"` // ssh隧道机器id
	EnableRecorder     int8   `json:"enableRecorder"`     // 是否启用终端回放记录
	TagId              uint64 `json:"tagId"`
	TagPath            string `json:"tagPath"`
}

func (m *Info) UseSshTunnel() bool {
	return m.SshTunnelMachineId > 0
}

// 获取记录日志的描述
func (m *Info) GetLogDesc() string {
	return fmt.Sprintf("Machine[id=%d, tag=%s, name=%s, ip=%s:%d]", m.Id, m.TagPath, m.Name, m.Ip, m.Port)
}

// 客户端信息
type Cli struct {
	machine *Info

	client     *ssh.Client  // ssh客户端
	sftpClient *sftp.Client // sftp客户端

	sshTunnelMachineId int
}

// 连接
func (c *Cli) connect() error {
	// 如果已经有client则直接返回
	if c.client != nil {
		return nil
	}
	m := c.machine
	sshClient, err := GetSshClient(m)
	if err != nil {
		return err
	}
	c.client = sshClient
	return nil
}

// 关闭client并从缓存中移除，如果使用隧道则也关闭
func (c *Cli) Close() {
	m := c.machine
	global.Log.Info(fmt.Sprintf("关闭机器客户端连接-> id: %d, name: %s, ip: %s", m.Id, m.Name, m.Ip))
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
	if c.sftpClient != nil {
		c.sftpClient.Close()
		c.sftpClient = nil
	}
	if c.sshTunnelMachineId > 0 {
		CloseSshTunnelMachine(c.sshTunnelMachineId, c.machine.Id)
	}
}

// 获取sftp client
func (c *Cli) GetSftpCli() *sftp.Client {
	if c.client == nil {
		if err := c.connect(); err != nil {
			panic(biz.NewBizErr("连接ssh失败：" + err.Error()))
		}
	}
	sftpclient := c.sftpClient
	// 如果sftpClient为nil，则连接
	if sftpclient == nil {
		sc, serr := sftp.NewClient(c.client)
		if serr != nil {
			panic(biz.NewBizErr("获取sftp client失败：" + serr.Error()))
		}
		sftpclient = sc
		c.sftpClient = sftpclient
	}

	return sftpclient
}

// 获取session
func (c *Cli) GetSession() (*ssh.Session, error) {
	if c.client == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}
	return c.client.NewSession()
}

// 执行shell
// @param shell shell脚本命令
// @return 返回执行成功或错误的消息
func (c *Cli) Run(shell string) (string, error) {
	session, err := c.GetSession()
	if err != nil {
		c.Close()
		return "", err
	}
	defer session.Close()
	buf, err := session.CombinedOutput(shell)
	if err != nil {
		return string(buf), err
	}
	return string(buf), nil
}

func (c *Cli) GetMachine() *Info {
	return c.machine
}

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
			if v.Value.(*Cli).sshTunnelMachineId == machineId {
				return true
			}
		}
		return false
	})
}

// 是否存在指定id的客户端连接
func HasCli(machineId uint64) bool {
	if _, ok := cliCache.Get(machineId); ok {
		return true
	}
	return false
}

// 删除指定机器客户端，并关闭客户端连接
func DeleteCli(id uint64) {
	cliCache.Delete(id)
}

// 从缓存中获取客户端信息，不存在则回调获取机器信息函数，并新建
func GetCli(machineId uint64, getMachine func(uint64) *Info) (*Cli, error) {
	cli, err := cliCache.ComputeIfAbsent(machineId, func(_ any) (any, error) {
		me := getMachine(machineId)
		err := IfUseSshTunnelChangeIpPort(me, getMachine)
		if err != nil {
			return nil, fmt.Errorf("ssh隧道连接失败: %s", err.Error())
		}
		c, err := newClient(me)
		if err != nil {
			CloseSshTunnelMachine(me.SshTunnelMachineId, me.Id)
			return nil, err
		}
		c.sshTunnelMachineId = me.SshTunnelMachineId
		return c, nil
	})

	if cli != nil {
		return cli.(*Cli), err
	}
	return nil, err
}

// 测试连接，使用传值的方式，而非引用。因为如果使用了ssh隧道，则ip和端口会变为本地映射地址与端口
func TestConn(me Info, getSshTunnelMachine func(uint64) *Info) error {
	originId := me.Id
	if originId == 0 {
		// 随机设置一个ip，如果使用了隧道则用于临时保存隧道
		me.Id = uint64(time.Now().Nanosecond())
	}

	err := IfUseSshTunnelChangeIpPort(&me, getSshTunnelMachine)
	biz.ErrIsNilAppendErr(err, "ssh隧道连接失败: %s")
	if me.UseSshTunnel() {
		defer CloseSshTunnelMachine(me.SshTunnelMachineId, me.Id)
	}
	sshClient, err := GetSshClient(&me)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	return nil
}

// 如果使用了ssh隧道，则修改机器ip port为暴露的ip port
func IfUseSshTunnelChangeIpPort(me *Info, getMachine func(uint64) *Info) error {
	if !me.UseSshTunnel() {
		return nil
	}
	sshTunnelMachine, err := GetSshTunnelMachine(me.SshTunnelMachineId, func(u uint64) *Info {
		return getMachine(u)
	})
	if err != nil {
		return err
	}
	exposeIp, exposePort, err := sshTunnelMachine.OpenSshTunnel(me.Id, me.Ip, me.Port)
	if err != nil {
		return err
	}
	// 修改机器ip地址
	me.Ip = exposeIp
	me.Port = exposePort
	return nil
}

func GetSshClient(m *Info) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: m.Username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 5 * time.Second,
	}

	if m.AuthMethod == entity.AuthCertAuthMethodPassword {
		config.Auth = []ssh.AuthMethod{ssh.Password(m.Password)}
	} else if m.AuthMethod == entity.MachineAuthMethodPublicKey {
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
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return sshClient, nil
}

// 根据机器信息创建客户端对象
func newClient(machine *Info) (*Cli, error) {
	if machine == nil {
		return nil, errors.New("机器不存在")
	}

	global.Log.Infof("[%s]机器连接：%s:%d", machine.Name, machine.Ip, machine.Port)
	cli := new(Cli)
	cli.machine = machine
	err := cli.connect()
	if err != nil {
		return nil, err
	}
	return cli, nil
}
