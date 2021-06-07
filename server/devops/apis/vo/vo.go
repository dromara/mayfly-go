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
	Id         *int64     `json:"id"`
	Name       *string    `json:"name"`
	Username   *string    `json:"username"`
	Ip         *string    `json:"ip"`
	Port       *int       `json:"port"`
	CreateTime *time.Time `json:"createTime"`
	Creator    *string    `json:"creator"`
	CreatorId  *int64     `json:"creatorId"`
	UpdateTime *time.Time `json:"updateTime"`
	Modifier   *string    `json:"modifier"`
	ModifierId *int64     `json:"modifierId"`
}

type MachineScriptVO struct {
	Id          *int64  `json:"id"`
	Name        *string `json:"name"`
	Script      *string `json:"script"`
	Type        *int    `json:"type"`
	Description *string `json:"description"`
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
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

type RoleVO struct {
	Id   *int64
	Name *string
}
