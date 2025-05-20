package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type MachineQuery struct {
	model.PageParam

	Id       uint64 `json:"id" form:"id"`
	Code     string `json:"code" form:"code"`
	Name     string `json:"name" form:"name"`
	Status   int8   `json:"status" form:"status"`
	Ip       string `json:"ip" form:"ip"` // IP地址
	TagPath  string `json:"tagPath" form:"tagPath"`
	Protocol int8   `json:"protocol" form:"protocol"`

	Keyword string `json:"keyword" form:"keyword"`
	Codes   []string
}

type AuthCertQuery struct {
	Id         uint64 `json:"id" form:"id"`
	Name       string `json:"name" form:"name"`
	AuthMethod string `json:"authMethod" form:"authMethod"`
}

type MachineTermOpQuery struct {
	StartCreateTime *time.Time
}
