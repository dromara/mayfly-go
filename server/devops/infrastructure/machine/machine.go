package machine

import (
	"errors"
	"fmt"
	"io"
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

// 机器客户端连接缓存，30分钟内没有访问则会被关闭
var cliCache = cache.NewTimedCache(30*time.Minute, 5*time.Second).
	WithUpdateAccessTime(true).
	OnEvicted(func(key string, value interface{}) {
		global.Log.Info(fmt.Sprintf("删除机器连接缓存 id: %s", key))
		value.(*Cli).Close()
	})

// 从缓存中获取客户端信息，不存在则回调获取机器信息函数，并新建
func GetCli(machineId uint64, getMachine func(uint64) *entity.Machine) (*Cli, error) {
	cli, err := cliCache.ComputeIfAbsent(fmt.Sprint(machineId), func(key string) (interface{}, error) {
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

	global.Log.Infof("机器连接：%s:%d", machine.Ip, machine.Port)
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
	if c.client != nil {
		c.client.Close()
	}
	if c.sftpClient != nil {
		c.sftpClient.Close()
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
		sc, serr := sftp.NewClient(c.client, sftp.MaxPacket(1<<15))
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

//执行带交互的命令
func (c *Cli) RunTerminal(shell string, stdout, stderr io.Writer) error {
	session, err := c.GetSession()
	if err != nil {
		return err
	}
	//defer session.Close()

	// fd := int(os.Stdin.Fd())
	// oldState, err := terminal.MakeRaw(fd)
	// if err != nil {
	// 	panic(err)
	// }
	// defer terminal.Restore(fd, oldState)

	// writer, err := session.StdinPipe()
	biz.ErrIsNilAppendErr(err, "获取session stdin 错误：%s")
	session.Stdout = stdout
	session.Stderr = stderr

	// termWidth, termHeight, err := terminal.GetSize(fd)
	// if err != nil {
	// 	panic(err)
	// }
	// Set up terminal modes
	// modes := ssh.TerminalModes{
	// 	ssh.ECHO:          1,     // enable echoing
	// 	ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
	// 	ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	// }

	// // Request pseudo terminal
	// if err := session.RequestPty("xterm-256color", 400, 800, modes); err != nil {
	// 	return err
	// }

	session.Shell()
	session.Wait()
	// writer.Write([]byte(shell))
	session.Run(shell)
	return nil
}

// 关闭指定机器的连接
func Close(id uint64) {
	if cli, ok := cliCache.Get(fmt.Sprint(id)); ok {
		cli.(*Cli).Close()
	}
}
