package mcm

import (
	"errors"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/spf13/cast"
	"golang.org/x/crypto/ssh"
)

// ErrRunTimeout 命令执行超时。由 RunWithTimeout 在主动关闭会话后返回，
// 调用方可用 errors.Is 识别后将结果标记为超时而非普通失败
var ErrRunTimeout = errors.New("the command execution timed out")

// Cli 机器客户端
type Cli struct {
	Info *MachineInfo // 机器信息

	sshClient  *ssh.Client  // ssh客户端
	sftpClient *sftp.Client // sftp客户端
}

/******************* pool.Conn impl *******************/

func (c *Cli) Ping() error {
	_, _, err := c.sshClient.SendRequest(
		"keepalive@openssh.com",
		true,
		nil,
	)
	return err
}

// Close 关闭client并从缓存中移除，如果使用隧道则也关闭
func (c *Cli) Close() error {
	m := c.Info
	logx.Debugf("close machine cli -> id=%d, name=%s, ip=%s", m.Id, m.Name, m.Ip)
	if c.sshClient != nil {
		c.sshClient.Close()
		c.sshClient = nil
	}
	if c.sftpClient != nil {
		c.sftpClient.Close()
		c.sftpClient = nil
	}

	CloseSshTunnel(m)

	return nil
}

// GetSftpCli 获取sftp client
func (c *Cli) GetSftpCli() (*sftp.Client, error) {
	if c.sshClient == nil {
		return nil, errorx.NewBiz("please connect to the machine client first")
	}
	sftpclient := c.sftpClient
	// 如果sftpClient为nil，则连接
	if sftpclient == nil {
		sc, serr := sftp.NewClient(c.sshClient)
		if serr != nil {
			return nil, errorx.NewBizf("failed to obtain the sftp client: %s", serr.Error())
		}
		sftpclient = sc
		c.sftpClient = sftpclient
	}

	return sftpclient, nil
}

// GetSession 获取session
func (c *Cli) GetSession() (*ssh.Session, error) {
	if c.sshClient == nil {
		return nil, errorx.NewBiz("please connect to the machine client first")
	}
	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, errorx.NewBizf("the acquisition session failed: %s, please try again later...", err.Error())
	}
	return session, nil
}

// Run 执行shell
//   - shell shell脚本命令
//
// @return 返回执行成功或错误的消息
func (c *Cli) Run(shell string) (string, error) {
	session, err := c.GetSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	// 将可能存在的windows换行符替换为linux格式
	buf, err := session.CombinedOutput(strings.ReplaceAll(shell, "\r\n", "\n"))
	if err != nil {
		return string(buf), err
	}
	return string(buf), nil
}

// RunWithTimeout 在限时内执行shell，超时则主动关闭session。
//
// CombinedOutput 会阻塞在会话读上，外层 select 超时只会让调用方先返回而 session 永远泄漏，
// 因此超时时必须主动 Close 会话令阻塞读立即返回（Close 幂等，与 defer 的收尾并发安全）。
// 归 mcm 的通用能力，不与任何具体业务场景（如批量执行）耦合
func (c *Cli) RunWithTimeout(timeout time.Duration, shell string) (string, error) {
	session, err := c.GetSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	timedOut := make(chan struct{})
	timer := time.AfterFunc(timeout, func() {
		close(timedOut)
		session.Close()
	})
	defer timer.Stop()

	// 将可能存在的windows换行符替换为linux格式
	buf, runErr := session.CombinedOutput(strings.ReplaceAll(shell, "\r\n", "\n"))
	// 超时触发的会话中断按超时错误上报，而非误导性的底层 EOF/reset
	select {
	case <-timedOut:
		return string(buf), ErrRunTimeout
	default:
	}
	if runErr != nil {
		return string(buf), runErr
	}
	return string(buf), nil
}

// GetAllStats 获取机器的所有状态信息
func (c *Cli) GetAllStats() *Stats {
	stats := new(Stats)
	res, err := c.Run(StatsShell)
	if err != nil {
		logx.Errorf("failed to execute machine [id=%d, name=%s] running status information script: %s", c.Info.Id, c.Info.Name, err.Error())
		return stats
	}

	infos := strings.Split(res, "-----")
	if len(infos) < 8 {
		return stats
	}
	getUptime(infos[0], stats)
	getHostname(infos[1], stats)
	getLoad(infos[2], stats)
	getMemInfo(infos[3], stats)
	getFSInfo(infos[4], stats)
	getInterfaces(infos[5], stats)
	getInterfaceInfo(infos[6], stats)
	getCPU(infos[7], stats)
	return stats
}

// GetUsers 读取/etc/passwd，获取系统所有用户信息
func (c *Cli) GetUsers() ([]*UserInfo, error) {
	res, err := c.Run("cat /etc/passwd")
	if err != nil {
		return nil, err
	}
	var users []*UserInfo
	userLines := strings.Split(res, "\n")
	for _, userLine := range userLines {
		if userLine == "" {
			continue
		}
		fields := strings.Split(userLine, ":")
		user := &UserInfo{
			Username: fields[0],
			UID:      cast.ToUint32(fields[2]),
			GID:      cast.ToUint32(fields[3]),
			HomeDir:  fields[5],
			Shell:    fields[6],
		}
		users = append(users, user)
	}

	return users, nil
}

// GetGroups 读取/etc/group，获取系统所有组信息
func (c *Cli) GetGroups() ([]*GroupInfo, error) {
	res, err := c.Run("cat /etc/group")
	if err != nil {
		return nil, err
	}

	var groups []*GroupInfo
	groupLines := strings.Split(res, "\n")
	for _, groupLine := range groupLines {
		if groupLine == "" {
			continue
		}
		fields := strings.Split(groupLine, ":")
		group := &GroupInfo{
			Groupname: fields[0],
			GID:       cast.ToUint32(fields[2]),
		}
		groups = append(groups, group)
	}

	return groups, nil
}
