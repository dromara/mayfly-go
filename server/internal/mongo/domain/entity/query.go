package entity

import "mayfly-go/pkg/model"

type MongoQuery struct {
	model.Model
	model.PageParam

	Code               string `json:"code" form:"code"`
	Name               string
	Keyword            string `json:"keyword" form:"keyword"`
	Uri                string
	SshTunnelMachineId uint64 // ssh隧道机器id
	TagPath            string `json:"tagPath" form:"tagPath"`

	Codes []string
}
