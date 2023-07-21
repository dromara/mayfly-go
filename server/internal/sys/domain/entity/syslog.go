package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// 系统操作日志
type SysLog struct {
	model.DeletedModel

	CreateTime time.Time `json:"createTime"`
	CreatorId  uint64    `json:"creatorId"`
	Creator    string    `json:"creator"`

	Type        int8   `json:"type"`
	Description string `json:"description"`
	ReqParam    string `json:"reqParam" gorm:"column:req_param;type:varchar(1000)"` // 请求参数
	Resp        string `json:"resp" gorm:"column:resp;type:varchar(1000)"`          // 响应结构
}

func (a *SysLog) TableName() string {
	return "t_sys_log"
}

const (
	SyslogTypeNorman int8 = 1 // 正常状态
	SyslogTypeError  int8 = 2 // 错误状态
)
