package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type Msg struct {
	model.DeletedModel

	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator" gorm:"size:50"`

	Type        int8   `json:"type"`
	Msg         string `json:"msg" gorm:"size:2000"`
	RecipientId int64  `json:"recipientId"` // 接收人id，-1为所有接收
}

func (a *Msg) TableName() string {
	return "t_sys_msg"
}
