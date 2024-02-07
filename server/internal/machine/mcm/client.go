package mcm

import (
	"fmt"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// 机器客户端
type Cli struct {
	Info *MachineInfo // 机器信息

	sshClient  *ssh.Client  // ssh客户端
	sftpClient *sftp.Client // sftp客户端
}

// 获取sftp client
func (c *Cli) GetSftpCli() (*sftp.Client, error) {
	if c.sshClient == nil {
		return nil, errorx.NewBiz("请先进行机器客户端连接")
	}
	sftpclient := c.sftpClient
	// 如果sftpClient为nil，则连接
	if sftpclient == nil {
		sc, serr := sftp.NewClient(c.sshClient)
		if serr != nil {
			return nil, errorx.NewBiz("获取sftp client失败: %s", serr.Error())
		}
		sftpclient = sc
		c.sftpClient = sftpclient
	}

	return sftpclient, nil
}

// 获取session
func (c *Cli) GetSession() (*ssh.Session, error) {
	if c.sshClient == nil {
		return nil, errorx.NewBiz("请先进行机器客户端连接")
	}
	session, err := c.sshClient.NewSession()
	if err != nil {
		// 获取session失败，则关闭cli，重试
		DeleteCli(c.Info.Id)
		logx.Errorf("获取机器客户端session失败: %s", err.Error())
		return nil, errorx.NewBiz("获取会话失败, 请重试...")
	}
	return session, nil
}

// 执行shell
// @param shell shell脚本命令
// @return 返回执行成功或错误的消息
func (c *Cli) Run(shell string) (string, error) {
	session, err := c.GetSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	buf, err := session.CombinedOutput(shell)
	if err != nil {
		return string(buf), err
	}
	return string(buf), nil
}

// 获取机器的所有状态信息
func (c *Cli) GetAllStats() *Stats {
	stats := new(Stats)
	res, err := c.Run(StatsShell)
	if err != nil {
		logx.Errorf("执行机器[id=%d, name=%s]运行状态信息脚本失败: %s", c.Info.Id, c.Info.Name, err.Error())
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

// 关闭client并从缓存中移除，如果使用隧道则也关闭
func (c *Cli) Close() {
	m := c.Info
	logx.Info(fmt.Sprintf("关闭机器客户端连接-> id: %d, name: %s, ip: %s", m.Id, m.Name, m.Ip))
	if c.sshClient != nil {
		c.sshClient.Close()
		c.sshClient = nil
	}
	if c.sftpClient != nil {
		c.sftpClient.Close()
		c.sftpClient = nil
	}

	var sshTunnelMachineId uint64
	if c.Info.SshTunnelMachine != nil {
		sshTunnelMachineId = c.Info.SshTunnelMachine.Id
	}
	if c.Info.TempSshMachineId != 0 {
		sshTunnelMachineId = c.Info.TempSshMachineId
	}
	if sshTunnelMachineId != 0 {
		logx.Infof("关闭机器的隧道信息: machineId=%d, sshTunnelMachineId=%d", c.Info.Id, sshTunnelMachineId)
		CloseSshTunnelMachine(int(c.Info.SshTunnelMachine.Id), c.Info.GetTunnelId())
	}
}
