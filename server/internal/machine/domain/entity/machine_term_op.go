package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type MachineTermOp struct {
	model.DeletedModel

	MachineId uint64 `json:"machineId"`
	Username  string `json:"username"`
	FileKey   string `json:"fileKey"`  // 文件key
	ExecCmds  string `json:"execCmds"` // 执行的命令

	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`
	EndTime    *time.Time `json:"endTime"`
}
