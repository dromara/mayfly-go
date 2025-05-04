package entity

import (
	"mayfly-go/pkg/model"
)

// 系统操作日志
type SysLog struct {
	model.CreateModel

	Type        int8   `json:"type" gorm:"not null;"`
	Description string `json:"description" gorm:"size:255;"`
	ReqParam    string `json:"reqParam" gorm:"type:text;"` // 请求参数
	Resp        string `json:"resp" gorm:"type:text;"`     // 响应结构
	Extra       string `json:"extra" gorm:"type:text;"`    // 日志额外信息
}

func (a *SysLog) TableName() string {
	return "t_sys_log"
}

const (
	SyslogTypeRunning int8 = -1 // 执行中
	SyslogTypeSuccess int8 = 1  // 正常状态
	SyslogTypeError   int8 = 2  // 错误状态
)
