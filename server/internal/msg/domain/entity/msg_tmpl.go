package entity

import (
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/model"
)

// MsgTmpl 消息模板
type MsgTmpl struct {
	model.Model
	model.ExtraData

	Name    string        `json:"name" gorm:"size:50;not null;"`   // 模板名称
	Code    string        `json:"code" gorm:"size:32;not null;"`   // 模板编码
	Title   string        `json:"title" gorm:"size:100;"`          // 标题
	Tmpl    string        `json:"tmpl" gorm:"size:2000;not null;"` // 消息模板
	MsgType msgx.MsgType  `json:"msgType" gorm:"not null;"`        // 消息类型
	Status  MsgTmplStatus `json:"status" gorm:"not null;"`         // 状态
	Remark  *string       `json:"remark" gorm:"size:200;"`         // 备注
}

func (a *MsgTmpl) TableName() string {
	return "t_msg_tmpl"
}

type MsgTmplStatus int8

const (
	TmplStatusEnable  MsgTmplStatus = 1  // 启用状态
	TmplStatusDisable MsgTmplStatus = -1 // 禁用状态
)

// MsgTmplChannel 消息模板渠道关联
type MsgTmplChannel struct {
	model.CreateModelNLD
	TmplId    uint64 `json:"tmplId" gorm:"not null;"`    // 模板id
	ChannelId uint64 `json:"channelId" gorm:"not null;"` // 渠道id
}

func (a *MsgTmplChannel) TableName() string {
	return "t_msg_tmpl_channel"
}
