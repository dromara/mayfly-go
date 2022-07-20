package entity

import (
	"fmt"
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Name      string `orm:"column(name)" json:"name"`
	Type      string `orm:"column(type)" json:"type"` // 类型，mysql oracle等
	Host      string `orm:"column(host)" json:"host"`
	Port      int    `orm:"column(port)" json:"port"`
	Network   string `orm:"column(network)" json:"network"`
	Username  string `orm:"column(username)" json:"username"`
	Password  string `orm:"column(password)" json:"-"`
	Database  string `orm:"column(database)" json:"database"`
	Params    string `json:"params"`
	ProjectId uint64
	Project   string
	EnvId     uint64
	Env       string

	EnableSshTunnel    int8   `orm:"column(enable_ssh_tunnel)" json:"enableSshTunnel"`        // 是否启用ssh隧道
	SshTunnelMachineId uint64 `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
}

// 获取数据库连接网络, 若没有使用ssh隧道，则直接返回。否则返回拼接的网络需要注册至指定dial
func (d Db) GetNetwork() string {
	network := d.Network
	if d.EnableSshTunnel == -1 {
		if network == "" {
			return "tcp"
		} else {
			return network
		}
	}
	return fmt.Sprintf("%s+ssh:%d", d.Type, d.SshTunnelMachineId)
}

const (
	DbTypeMysql    = "mysql"
	DbTypePostgres = "postgres"
)
