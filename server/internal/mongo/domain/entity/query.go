package entity

import "mayfly-go/pkg/model"

type MongoQuery struct {
	model.Model

	Name               string
	Uri                string
	SshTunnelMachineId uint64 // ssh隧道机器id
	TagPath            string `json:"tagPath" form:"tagPath"`

	Codes []string
}
