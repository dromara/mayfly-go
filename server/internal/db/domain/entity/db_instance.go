package entity

import (
	"errors"
	"fmt"
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/model"
)

type DbInstance struct {
	model.Model

	Name               string `json:"name"`
	Type               string `json:"type"` // 类型，mysql oracle等
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Network            string `json:"network"`
	Sid                string `json:"sid"`
	Username           string `json:"username"`
	Password           string `json:"-"`
	Params             string `json:"params"`
	Remark             string `json:"remark"`
	SshTunnelMachineId int    `json:"sshTunnelMachineId"` // ssh隧道机器id
}

func (d *DbInstance) TableName() string {
	return "t_db_instance"
}

// 获取数据库连接网络, 若没有使用ssh隧道，则直接返回。否则返回拼接的网络需要注册至指定dial
func (d *DbInstance) GetNetwork() string {
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

func (d *DbInstance) PwdEncrypt() error {
	// 密码替换为加密后的密码
	password, err := utils.PwdAesEncrypt(d.Password)
	if err != nil {
		return errors.New("加密数据库密码失败")
	}
	d.Password = password
	return nil
}

func (d *DbInstance) PwdDecrypt() error {
	// 密码替换为解密后的密码
	password, err := utils.PwdAesDecrypt(d.Password)
	if err != nil {
		return errors.New("解密数据库密码失败")
	}
	d.Password = password
	return nil
}
