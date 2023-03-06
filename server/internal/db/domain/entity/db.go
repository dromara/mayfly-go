package entity

import (
	"fmt"
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Name               string `orm:"column(name)" json:"name"`
	Type               string `orm:"column(type)" json:"type"` // 类型，mysql oracle等
	Host               string `orm:"column(host)" json:"host"`
	Port               int    `orm:"column(port)" json:"port"`
	Network            string `orm:"column(network)" json:"network"`
	Username           string `orm:"column(username)" json:"username"`
	Password           string `orm:"column(password)" json:"-"`
	Database           string `orm:"column(database)" json:"database"`
	Params             string `json:"params"`
	Remark             string `json:"remark"`
	TagId              uint64
	TagPath            string
	SshTunnelMachineId int `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
}

// 获取数据库连接网络, 若没有使用ssh隧道，则直接返回。否则返回拼接的网络需要注册至指定dial
func (d *Db) GetNetwork() string {
	network := d.Network
	if d.SshTunnelMachineId <= 0 {
		if network == "" {
			return "tcp"
		} else {
			return network
		}
	}
	return fmt.Sprintf("%s+ssh:%d", d.Type, d.SshTunnelMachineId)
}

func (d *Db) PwdEncrypt() {
	// 密码替换为加密后的密码
	d.Password = utils.PwdAesEncrypt(d.Password)
}

func (d *Db) PwdDecrypt() {
	// 密码替换为解密后的密码
	d.Password = utils.PwdAesDecrypt(d.Password)
}

const (
	DbTypeMysql    = "mysql"
	DbTypePostgres = "postgres"
)
