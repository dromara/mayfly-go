package entity

import (
	"mayfly-go/pkg/model"
)

// 系统操作日志
type SysLog struct {
	model.CreateModel

	Type        int8   `json:"type"`
	Description string `json:"description"`
	ReqParam    string `json:"reqParam" gorm:"column:req_param;type:varchar(1000)"` // 请求参数
	Resp        string `json:"resp" gorm:"column:resp;type:varchar(10000)"`         // 响应结构
	Extra       string `json:"extra"`                                               // 日志额外信息
}

func (a *SysLog) TableName() string {
	return "t_sys_log"
}

const (
	SyslogTypeRunning int8 = -1 // 执行中
	SyslogTypeSuccess int8 = 1  // 正常状态
	SyslogTypeError   int8 = 2  // 错误状态
)
