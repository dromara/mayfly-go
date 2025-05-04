package entity

import (
	"mayfly-go/pkg/model"
)

// MsgTmplBiz 消息模板关联业务信息
type MsgTmplBiz struct {
	model.Model

	TmplId  uint64 `json:"tmplId" gorm:"not null;"`          // 模板id
	BizId   uint64 `json:"bizId" gorm:"not null;"`           // 业务id
	BizType string `json:"bizType" gorm:"size:32;not null;"` // 业务类型
}

func (a *MsgTmplBiz) TableName() string {
	return "t_msg_tmpl_biz"
}
