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
}

type RoleVO struct {
	Id   *int64
	Name *string
}
