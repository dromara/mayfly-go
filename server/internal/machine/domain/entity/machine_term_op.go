package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type MachineTermOp struct {
	model.DeletedModel

	MachineId      uint64 `json:"machineId"`
	Username       string `json:"username"`
	RecordFilePath string `json:"recordFilePath"` // 回放文件路径

	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`
	EndTime    *time.Time `json:"endTime"`
}
