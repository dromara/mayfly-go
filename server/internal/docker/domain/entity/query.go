package entity

import "mayfly-go/pkg/model"

type ContainerQuery struct {
	model.PageParam

	Id      uint64 `json:"id" form:"id"`
	Code    string `json:"code" form:"code"`
	Name    string `json:"name" form:"name"`
	Addr    string `json:"addr" form:"addr"`
	TagPath string `json:"tagPath" form:"tagPath"`

	Keyword string `json:"keyword" form:"keyword"`
	Codes   []string
}
