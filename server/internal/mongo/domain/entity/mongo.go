package entity

import "mayfly-go/pkg/model"

type Mongo struct {
	model.Model

	Name               string `orm:"column(name)" json:"name"`
	Uri                string `orm:"column(uri)" json:"uri"`
	EnableSshTunnel    int8   `orm:"column(enable_ssh_tunnel)" json:"enableSshTunnel"`        // 是否启用ssh隧道
	SshTunnelMachineId uint64 `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
	ProjectId          uint64 `json:"projectId"`
	Project            string `json:"project"`
	EnvId              uint64 `json:"envId"`
	Env                string `json:"env"`
}
