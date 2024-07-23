package entity

import (
	"fmt"
	"mayfly-go/pkg/model"
)

type DbInstance struct {
	model.Model

	Code               string  `json:"code"`
	Name               string  `json:"name"`
	Type               string  `json:"type"` // 类型，mysql oracle等
	Host               string  `json:"host"`
	Port               int     `json:"port"`
	Network            string  `json:"network"`
	Extra              *string `json:"extra"`  // 连接需要的其他额外参数（json格式）, 如oracle需要sid等
	Params             *string `json:"params"` // 使用指针类型，可更新为零值（空字符串）
	Remark             *string `json:"remark"`
	SshTunnelMachineId int     `json:"sshTunnelMachineId"` // ssh隧道机器id
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
