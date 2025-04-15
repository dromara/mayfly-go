package form

import (
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/model"
)

type MsgChannel struct {
	model.ExtraData

	Id     uint64 `json:"id"`
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`
	Url    string `json:"url"`
	Remark string `json:"remark"`
	Status int8   `json:"status" binding:"required"`
}

type MsgTmpl struct {
	model.ExtraData

	Id         uint64       `json:"id"`
	Name       string       `json:"name" binding:"required"`
	Title      string       `json:"title"`
	Tmpl       string       `json:"tmpl" binding:"required"`
	MsgType    msgx.MsgType `json:"msgType" binding:"required"`
	Remark     string       `json:"remark"`
	Status     int8         `json:"status" binding:"required"`
	ChannelIds []uint64     `json:"channelIds"`
}

type SendMsg struct {
	Parmas      string   `json:"params"`
	ReceiverIds []uint64 `json:"receiverIds"`
}
