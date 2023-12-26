package form

type MachineForm struct {
	Id   uint64 `json:"id"`
	Name string `json:"name" binding:"required"`
	Ip   string `json:"ip" binding:"required"`   // IP地址
	Port int    `json:"port" binding:"required"` // 端口号

	// 资产授权凭证信息列表
	AuthCertId int      `json:"authCertId"`
	TagId      []uint64 `json:"tagId" binding:"required"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`

	Remark             string `json:"remark"`
	SshTunnelMachineId int    `json:"sshTunnelMachineId"` // ssh隧道机器id
	EnableRecorder     int8   `json:"enableRecorder"`     // 是否启用终端回放记录
}

type MachineRunForm struct {
	MachineId int64  `json:"machineId" binding:"required"`
	Cmd       string `json:"cmd" binding:"required"`
}

type MachineFileForm struct {
	Id        uint64 `json:"id"`
	Name      string `json:"name" binding:"required"`
	MachineId uint64 `json:"machineId" binding:"required"`
	Type      int    `json:"type" binding:"required"`
	Path      string `json:"path" binding:"required"`
}

type MachineScriptForm struct {
	Id          uint64 `json:"id"`
	Name        string `json:"name" binding:"required"`
	MachineId   uint64 `json:"machineId" binding:"required"`
	Type        int    `json:"type" binding:"required"`
	Description string `json:"description" binding:"required"`
	Params      string `json:"params"`
	Script      string `json:"script" binding:"required"`
}

type MachineCreateFileForm struct {
	Path string `json:"path" binding:"required"`
	Type string `json:"type" binding:"required"`
}

type MachineFileUpdateForm struct {
	Content string `json:"content" binding:"required"`
	Id      uint64 `json:"id" binding:"required"`
	Path    string `json:"path" binding:"required"`
}

type MachineFileOpForm struct {
	Path   []string `json:"path" binding:"required"`
	ToPath string   `json:"toPath"`
}

type MachineFileRename struct {
	Oldname string `json:"oldname" binding:"required"`
	Newname string `json:"newname"  binding:"required"`
}

// 授权凭证
type AuthCertForm struct {
	Id         uint64 `json:"id"`
	Name       string `json:"name" binding:"required"`
	AuthMethod int8   `json:"authMethod" binding:"required"` // 1.密码 2.秘钥
	Username   string `json:"username"`
	Password   string `json:"password"`   // 密码or私钥
	Passphrase string `json:"passphrase"` // 私钥口令
	Remark     string `json:"remark"`
}

// 机器记录任务
type MachineCronJobForm struct {
	Id              uint64   `json:"id"`
	Name            string   `json:"name" binding:"required"`
	Cron            string   `json:"cron" binding:"required"` // cron
	Script          string   `json:"script" binding:"required"`
	Status          int      `json:"status" binding:"required"`
	SaveExecResType int      `json:"saveExecResType" binding:"required"`
	MachineIds      []uint64 `json:"machineIds"`
	Remark          string   `json:"remark"`
}
