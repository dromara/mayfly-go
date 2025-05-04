package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type MachineTermOp struct {
	model.DeletedModel

	MachineId uint64 `json:"machineId" gorm:"not null;comment:机器id"`    // 机器id
	Username  string `json:"username" gorm:"size:60;comment:登录用户名"`     // 登录用户名
	FileKey   string `json:"fileKey" gorm:"size:36;comment:文件"`         // 文件key
	ExecCmds  string `json:"execCmds" gorm:"type:text;comment:执行的命令记录"` // 执行的命令

	CreateTime *time.Time `json:"createTime" gorm:"not null;comment:创建时间"` // 创建时间
	CreatorId  uint64     `json:"creatorId" gorm:"comment:创建人ID"`
	Creator    string     `json:"creator" gorm:"size:50;comment:创建人"` // 创建人
	EndTime    *time.Time `json:"endTime" gorm:"comment:结束时间"`        // 结束时间
}
