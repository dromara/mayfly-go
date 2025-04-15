package dto

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/model"
)

type MsgTmplSave struct {
	model.ExtraData

	Id      uint64               `json:"id"`
	Name    string               `json:"name"`
	Remark  string               `json:"remark"`
	Status  entity.MsgTmplStatus `json:"status" `
	Title   string               `json:"title"`
	Tmpl    string               `json:"type"`
	MsgType msgx.MsgType         `json:"msgType"`

	ChannelIds []uint64 `json:"channelIds"`
}

// MsgTmplBizSave 消息模板关联业务信息
type MsgTmplBizSave struct {
	TmplId  uint64 // 消息模板id
	BizId   uint64 // 业务id
	BizType string
}

// BizMsgTmplSend 业务消息模板发送消息
type BizMsgTmplSend struct {
	BizId       uint64 // 业务id
	BizType     string
	Params      map[string]any // 模板占位符参数
	ReceiverIds []uint64       // 接收人id
}
