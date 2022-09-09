package vo

import "time"

type AccountVO struct {
	//models.BaseModel
	Id         *int64  `json:"id"`
	Username   *string `json:"username"`
	CreateTime *string `json:"createTime"`
	Creator    *string `json:"creator"`
	CreatorId  *int64  `json:"creatorId"`
	// Role       *RoleVO `json:"roles"`
	//Status   int8   `json:"status"`
}

type MachineVO struct {
	//models.BaseModel
	Id                 *uint64    `json:"id"`
	ProjectId          uint64     `json:"projectId"`
	ProjectName        string     `json:"projectName"`
	Name               *string    `json:"name"`
	Username           *string    `json:"username"`
	Ip                 *string    `json:"ip"`
	Port               *int       `json:"port"`
	AuthMethod         *int8      `json:"authMethod"`
	Status             *int8      `json:"status"`
	EnableSshTunnel    *int8      `json:"enableSshTunnel"`    // 是否启用ssh隧道
	SshTunnelMachineId *uint64    `json:"sshTunnelMachineId"` // ssh隧道机器id
	CreateTime         *time.Time `json:"createTime"`
	Creator            *string    `json:"creator"`
	CreatorId          *int64     `json:"creatorId"`
	UpdateTime         *time.Time `json:"updateTime"`
	Modifier           *string    `json:"modifier"`
	ModifierId         *int64     `json:"modifierId"`
	HasCli             bool       `json:"hasCli" gorm:"-"`
	Remark             *string    `json:"remark"`
	EnableRecorder     int8       `json:"enableRecorder"`
}

type MachineScriptVO struct {
	Id          *int64  `json:"id"`
	Name        *string `json:"name"`
	Script      *string `json:"script"`
	Type        *int    `json:"type"`
	Description *string `json:"description"`
	Params      *string `json:"params"`
	MachineId   *uint64 `json:"machineId"`
}

type MachineFileVO struct {
	Id        *int64  `json:"id"`
	Name      *string `json:"name"`
	Path      *string `json:"path"`
	Type      *int    `json:"type"`
	MachineId *uint64 `json:"machineId"`
}

type MachineFileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
}

type RoleVO struct {
	Id   *int64
	Name *string
}
