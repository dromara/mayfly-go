package entity

import (
	"fmt"
	"mayfly-go/pkg/model"
)

type EsInstance struct {
	model.Model

	Code               string  `json:"code" gorm:"size:32;not null;"`
	Name               string  `json:"name" gorm:"size:32;not null;"`
	Host               string  `json:"host" gorm:"size:255;not null;"`
	Port               int     `json:"port"`
	Network            string  `json:"network" gorm:"size:20;"`
	Version            string  `json:"version" gorm:"size:50;"`
	Remark             *string `json:"remark" gorm:"size:255;"`
	SshTunnelMachineId int     `json:"sshTunnelMachineId"` // ssh隧道机器id
}

func (d *EsInstance) TableName() string {
	return "t_es_instance"
}

// 获取es连接网络, 若没有使用ssh隧道，则直接返回。否则返回拼接的网络需要注册至指定dial
func (d *EsInstance) GetNetwork() string {
	network := d.Network
	if d.SshTunnelMachineId <= 0 {
		if network == "" {
			return "tcp"
		} else {
			return network
		}
	}
	return fmt.Sprintf("es+ssh:%d", d.SshTunnelMachineId)
}
