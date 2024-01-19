package form

type InstanceForm struct {
	Id                 uint64 `json:"id"`
	Name               string `binding:"required" json:"name"`
	Type               string `binding:"required" json:"type"` // 类型，mysql oracle等
	Host               string `binding:"required" json:"host"`
	Port               int    `json:"port"`
	Sid                string `json:"sid"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	Params             string `json:"params"`
	Remark             string `json:"remark"`
	SshTunnelMachineId int    `json:"sshTunnelMachineId"`
}
