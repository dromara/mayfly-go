package form

type MachineForm struct {
	Id   uint64 `json:"id"`
	Name string `json:"name" binding:"required"`
	Ip   string `json:"ip" binding:"required"`   // IP地址
	Port int    `json:"port" binding:"required"` // 端口号

	// 资产授权凭证信息列表
	AuthCertId int    `json:"authCertId"`
	TagId      uint64 `json:"tagId"`
	TagPath    string `json:"tagPath" binding:"required"`
	Username   string `json:"username"`
	Password   string `json:"password"`

	Remark             string `json:"remark"`
	SshTunnelMachineId int    `json:"sshTunnelMachineId"` // ssh隧道机器id
	EnableRecorder     int8   `json:"enableRecorder"`     // 是否启用终端回放记录
}

type MachineRunForm struct {
	MachineId int64  `binding:"required"`
	Cmd       string `binding:"required"`
}

type MachineFileForm struct {
	Id        uint64
	Name      string `binding:"required"`
	MachineId uint64 `binding:"required"`
	Type      int    `binding:"required"`
	Path      string `binding:"required"`
}

type MachineScriptForm struct {
	Id          uint64
	Name        string `binding:"required"`
	MachineId   uint64 `binding:"required"`
	Type        int    `binding:"required"`
	Description string `binding:"required"`
	Params      string
	Script      string `binding:"required"`
}

type MachineCreateFileForm struct {
	Path string `binding:"required"`
	Type string `binding:"required"`
}

type MachineFileUpdateForm struct {
	Content string `binding:"required"`
	Id      uint64 `binding:"required"`
	Path    string `binding:"required"`
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

// 资产授权凭证信息
type AssetAuthCertForm struct {
	AuthCertId uint64 `json:"authCertId"`
	TagId      uint64 `json:"tagId"`
	TagPath    string `json:"tagPath" binding:"required"`
}
