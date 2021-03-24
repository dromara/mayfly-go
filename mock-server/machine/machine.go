package machine

import (
	"errors"
	"fmt"
	"io"
	"mayfly-go/base/biz"
	"mayfly-go/base/utils"
	"mayfly-go/mock-server/models"
	"net"
	"os"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// 客户端信息
type Cli struct {
	machine *models.Machine
	// ssh客户端
	client *ssh.Client
}

// 客户端缓存
var clientCache sync.Map
var mutex sync.Mutex

// 从缓存中获取客户端信息，不存在则查库，并新建
func GetCli(machineIp string) (*Cli, error) {
	mutex.Lock()
	defer mutex.Unlock()
	load, ok := clientCache.Load(machineIp)
	if ok {
		return load.(*Cli), nil
	}

	cli, err := newClient(models.GetMachineByIp(machineIp))
	if err != nil {
		return nil, err
	}
	clientCache.LoadOrStore(machineIp, cli)
	return cli, nil
}

//根据机器信息创建客户端对象
func newClient(machine *models.Machine) (*Cli, error) {
	if machine == nil {
		return nil, errors.New("机器不存在")
	}

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
func TestConn(m *models.Machine) (*ssh.Client, error) {
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
		return nil, err
	}
	return sshClient, nil
}

// 关闭client和并从缓存中移除
func (c *Cli) Close() {
	if c.client != nil {
		c.client.Close()
	}
	if utils.StrLen(c.machine.Ip) > 0 {
		clientCache.Delete(c.machine.Ip)
	}
}

// 获取sftp client
func (c *Cli) GetSftpCli() *sftp.Client {
	if c.client == nil {
		if err := c.connect(); err != nil {
			panic(biz.NewBizErr("连接ssh失败：" + err.Error()))
		}
	}
	client, serr := sftp.NewClient(c.client, sftp.MaxPacket(1<<15))
	if serr != nil {
		panic(biz.NewBizErr("获取sftp client失败：" + serr.Error()))
	}
	return client
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

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(fd, oldState)

	session.Stdout = stdout
	session.Stderr = stderr
	session.Stdin = os.Stdin

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		panic(err)
	}
	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		return err
	}

	return session.Run(shell)
}
