package entity

import (
	"time"
)

type Msg struct {
	Id         uint64     `json:"id"`
	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`

	Type        int    `json:"type"`
	Msg         string `json:"msg"`
	RecipientId int64  `json:"recipientId"` // 接受者id
}

func (a *Msg) TableName() string {
	return "t_sys_msg"
}
