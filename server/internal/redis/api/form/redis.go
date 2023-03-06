package form

type Redis struct {
	Id                 uint64
	Name               string `json:"name"`
	Host               string `binding:"required" json:"host"`
	Password           string `json:"password"`
	Mode               string `json:"mode"`
	Db                 string `json:"db"`
	SshTunnelMachineId int    `json:"sshTunnelMachineId"` // ssh隧道机器id
	TagId              uint64 `binding:"required" json:"tagId"`
	TagPath            string `json:"tagPath"`
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
