package entity

import "mayfly-go/pkg/model"

type RedisQuery struct {
	model.PageParam

	Id                 uint64 `form:"id"`
	Code               string `json:"code" form:"code"`
	Name               string `orm:"column(name)" json:"name" form:"name"`
	Host               string `orm:"column(host)" json:"host" form:"host"`
	Keyword            string `json:"keyword" form:"keyword"`
	SshTunnelMachineId int    `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id

	Codes   []string
	TagPath string `form:"tagPath"`
}
