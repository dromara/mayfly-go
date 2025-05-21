package entity

import "mayfly-go/pkg/model"

// InstanceQuery 数据库实例查询
type InstanceQuery struct {
	model.PageParam

	Id      uint64 `json:"id" form:"id"`
	Name    string `json:"name" form:"name"`
	Code    string `json:"code" form:"code"`
	Host    string `json:"host" form:"host"`
	TagPath string `json:"tagPath" form:"tagPath"`
	Keyword string `json:"keyword" form:"keyword"`
	Codes   []string
}
