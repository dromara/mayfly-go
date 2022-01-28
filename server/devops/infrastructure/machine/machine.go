package machine

import (
	"errors"
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/cache"
	"mayfly-go/base/global"
	"mayfly-go/server/devops/domain/entity"
	"net"
	"time"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

// 客户端信息
type Cli struct {
	machine *entity.Machine
	// ssh客户端
	client *ssh.Client

	sftpClient *sftp.Client
}

// 机器客户端连接缓存，45分钟内没有访问则会被关闭
var cliCache = cache.NewTimedCache(45*time.Minute, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key interface{}, value interface{}) {
		value.(*Cli).Close()
	})

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
func GetCli(machineId uint64, getMachine func(uint64) *entity.Machine) (*Cli, error) {
	cli, err := cliCache.ComputeIfAbsent(machineId, func(key interface{}) (interface{}, error) {
		c, err := newClient(getMachine(machineId))
		if err != nil {
			return nil, err
		}
		return c, nil
	})

	if cli != nil {
		return cli.(*Cli), err
	}
	return nil, err
}

//根据机器信息创建客户端对象
func newClient(machine *entity.Machine) (*Cli, error) {
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

//连接
func (c *Cli) connect() error {
	// 如果已经有client则直接返回
	if c.client != nil {
		return nil
	}
	m := c.machine
	config := ssh.ClientConfig{
		User: m.Username,
		Auth: []ssh.AuthMethod{ssh.Password(m.Password)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", m.Ip, m.Port)
	sshClient, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		return err
	}
	c.client = sshClient
	return nil
}

// 测试连接
func TestConn(m *entity.Machine) error {
	config := ssh.ClientConfig{
		User: m.Username,
		Auth: []ssh.AuthMethod{ssh.Password(m.Password)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", m.Ip, m.Port)
	sshClient, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		return err
	}
	defer sshClient.Close()
	return nil
}

// 关闭client和并从缓存中移除
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

//执行shell
//@param shell shell脚本命令
func (c *Cli) Run(shell string) (*string, error) {
	session, err := c.GetSession()
	if err != nil {
		c.Close()
		return nil, err
	}
	defer session.Close()
	buf, rerr := session.CombinedOutput(shell)
	if rerr != nil {
		return nil, rerr
	}
	res := string(buf)
	return &res, nil
}
