package form

type Redis struct {
	Id                 uint64
	Host               string `binding:"required" json:"host"`
	Password           string `json:"password"`
	Mode               string `json:"mode"`
	Db                 int    `json:"db"`
	EnableSshTunnel    int8   `json:"enableSshTunnel"`    // 是否启用ssh隧道
	SshTunnelMachineId uint64 `json:"sshTunnelMachineId"` // ssh隧道机器id
	ProjectId          uint64 `binding:"required" json:"projectId"`
	Project            string `json:"project"`
	Env                string `json:"env"`
	EnvId              uint64 `binding:"required" json:"envId"`
	Remark             string `json:"remark"`
}

type KeyInfo struct {
	Key   string `binding:"required" json:"key"`
	Timed int64
}

type StringValue struct {
	KeyInfo
	Value interface{} `binding:"required" json:"value"`
}

type HashValue struct {
	KeyInfo
	Value []map[string]interface{} `binding:"required" json:"value"`
}

type SetValue struct {
	KeyInfo
	Value []interface{} `binding:"required" json:"value"`
}

type ListValue struct {
	KeyInfo
	Value []interface{} `binding:"required" json:"value"`
}

// list lset命令参数入参
type ListSetValue struct {
	Key   string `binding:"required" json:"key"`
	Index int64
	Value interface{} `binding:"required" json:"value"`
}

type RedisScanForm struct {
	Cursor map[string]uint64 `json:"cursor"`
	Match  string            `json:"match"`
	Count  int64             `json:"count"`
}
