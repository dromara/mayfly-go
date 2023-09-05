package entity

import (
	"fmt"
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/model"
)

type Instance struct {
	model.Model

	Name               string `orm:"column(name)" json:"name"`
	Type               string `orm:"column(type)" json:"type"` // 类型，mysql oracle等
	Host               string `orm:"column(host)" json:"host"`
	Port               int    `orm:"column(port)" json:"port"`
	Network            string `orm:"column(network)" json:"network"`
	Username           string `orm:"column(username)" json:"username"`
	Password           string `orm:"column(password)" json:"-"`
	Params             string `orm:"column(params)" json:"params"`
	Remark             string `orm:"column(remark)" json:"remark"`
	SshTunnelMachineId int    `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
}

func (d *Instance) TableName() string {
	return "t_db_instance"
}

// 获取数据库连接网络, 若没有使用ssh隧道，则直接返回。否则返回拼接的网络需要注册至指定dial
func (d *Instance) GetNetwork() string {
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

func (d *Instance) PwdEncrypt() {
	// 密码替换为加密后的密码
	d.Password = utils.PwdAesEncrypt(d.Password)
}

func (d *Instance) PwdDecrypt() {
	// 密码替换为解密后的密码
	d.Password = utils.PwdAesDecrypt(d.Password)
}

const (
	DbTypeMysql    = "mysql"
	DbTypePostgres = "postgres"
)
