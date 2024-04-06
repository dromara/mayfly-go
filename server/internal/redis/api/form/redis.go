package form

type Redis struct {
	Id                 uint64   `json:"id"`
	Code               string   `json:"code" binding:"required"`
	Name               string   `json:"name"`
	Host               string   `json:"host" binding:"required"`
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	Mode               string   `json:"mode"`
	Db                 string   `json:"db"`
	SshTunnelMachineId int      `json:"sshTunnelMachineId"` // ssh隧道机器id
	TagId              []uint64 `binding:"required" json:"tagId"`
	Remark             string   `json:"remark"`
	FlowProcdefKey     string   `json:"flowProcdefKey"` // 审批流-流程定义key（有值则说明关键操作需要进行审批执行）,使用指针为了方便更新空字符串(取消流程审批)
}

type KeyInfo struct {
	Key   string `binding:"required" json:"key"`
	Timed int64  `json:"timed"`
}

type RedisScanForm struct {
	Cursor map[string]uint64 `json:"cursor"`
	Match  string            `json:"match"`
	Count  int64             `json:"count"`
}

type ScanForm struct {
	Key    string `json:"key"`
	Cursor uint64 `json:"cursor"`
	Match  string `json:"match"`
	Count  int64  `json:"count"`
}

type RunCmdForm struct {
	Id     uint64 `json:"id"`
	Db     int    `json:"db"`
	Cmd    []any  `json:"cmd"`
	Remark string `json:"remark"`
}
