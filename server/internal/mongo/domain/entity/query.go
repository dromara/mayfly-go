package entity

import "mayfly-go/pkg/model"

type MongoQuery struct {
	model.Model

	Name               string
	Uri                string
	SshTunnelMachineId uint64 // ssh隧道机器id
	TagId              uint64 `json:"tagId"`
	TagPath            string `json:"tagPath"`

	TagIds      []uint64
	TagPathLike string
}
