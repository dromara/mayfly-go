package entity

import "time"

type MachineQuery struct {
	Ids     string `json:"ids" form:"ids"`
	Name    string `json:"name" form:"name"`
	Status  int8   `json:"status" form:"status"`
	Ip      string `json:"ip" form:"ip"` // IP地址
	TagPath string `json:"tagPath" form:"tagPath"`

	Codes []string
}

type AuthCertQuery struct {
	Id         uint64 `json:"id" form:"id"`
	Name       string `json:"name" form:"name"`
	AuthMethod string `json:"authMethod" form:"authMethod"` // IP地址
}

type MachineTermOpQuery struct {
	StartCreateTime *time.Time
}
