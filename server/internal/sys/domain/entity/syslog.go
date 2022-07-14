package entity

import "time"

// 系统操作日志
type Syslog struct {
	Id         uint64    `json:"id"`
	CreateTime time.Time `json:"createTime"`
	CreatorId  uint64    `json:"creatorId"`
	Creator    string    `json:"creator"`

	Type        int8   `json:"type"`
	Description string `json:"description"`
	ReqParam    string `json:"reqParam"` // 请求参数
	Resp        string `json:"resp"`     // 响应结构
}

func (a *Syslog) TableName() string {
	return "t_sys_log"
}

const (
	SyslogTypeNorman int8 = 1 // 正常状态
	SyslogTypeError  int8 = 2 // 错误状态
)
