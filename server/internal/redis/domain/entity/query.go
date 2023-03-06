package entity

import "mayfly-go/pkg/model"

type RedisQuery struct {
	model.Model

	Name               string `orm:"column(name)" json:"name"`
	Host               string `orm:"column(host)" json:"host"`
	Mode               string `json:"mode"`
	Password           string `orm:"column(password)" json:"-"`
	Db                 string `orm:"column(database)" json:"db"`
	SshTunnelMachineId int    `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
	Remark             string
	TagId              uint64

	TagIds      []uint64
	TagPathLike string
}
