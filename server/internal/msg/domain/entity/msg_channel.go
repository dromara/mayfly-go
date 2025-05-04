package entity

import (
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/model"
)

type MsgChannel struct {
	model.Model
	model.ExtraData

	Name   string           `json:"name" gorm:"size:50;not null;"` // 渠道名称
	Code   string           `json:"code" gorm:"size:50;not null;"` // 渠道编码
	Type   msgx.ChannelType `json:"type" gorm:"size:30;not null;"` // 渠道类型
	Url    string           `json:"url" gorm:"size:200;"`          // 渠道url
	Status MsgChannelStatus `json:"status" gorm:"not null;"`       // 状态
	Remark *string          `json:"remark" gorm:"size:200;"`       // 备注
}

func (a *MsgChannel) TableName() string {
	return "t_msg_channel"
}

type MsgChannelStatus int8

const (
	ChannelStatusEnable  MsgChannelStatus = 1  // 启用状态
	ChannelStatusDisable MsgChannelStatus = -1 // 禁用状态
)
