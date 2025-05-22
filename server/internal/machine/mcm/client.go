package mcm

import (
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"strings"

	"github.com/may-fly/cast"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Cli 机器客户端
type Cli struct {
	Info *MachineInfo // 机器信息

	sshClient  *ssh.Client  // ssh客户端
	sftpClient *sftp.Client // sftp客户端
}

/******************* pool.Conn impl *******************/

func (c *Cli) Ping() error {
	_, _, err := c.sshClient.SendRequest("ping", true, nil)
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

	var sshTunnelMachineId uint64
	if m.SshTunnelMachine != nil {
		sshTunnelMachineId = m.SshTunnelMachine.Id
	}
	if m.TempSshMachineId != 0 {
		sshTunnelMachineId = m.TempSshMachineId
	}
	if sshTunnelMachineId != 0 {
		logx.Debugf("close machine ssh tunnel -> machineId=%d, sshTunnelMachineId=%d", m.Id, sshTunnelMachineId)
		CloseSshTunnelMachine(sshTunnelMachineId, m.GetTunnelId())
	}

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
			return nil, errorx.NewBiz("failed to obtain the sftp client: %s", serr.Error())
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
		logx.Errorf("failed to retrieve the machine client session: %s", err.Error())
		return nil, errorx.NewBiz("the acquisition session failed, please try again later...")
	}
	return session, nil
}

// Run 执行shell
// @param shell shell脚本命令
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
