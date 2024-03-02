package entity

type RedisQuery struct {
	Id   uint64 `form:"id"`
	Name string `orm:"column(name)" json:"name" form:"name"`
	Host string `orm:"column(host)" json:"host" form:"host"`

	SshTunnelMachineId int `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id

	Codes   []string
	TagPath string `form:"tagPath"`
}
