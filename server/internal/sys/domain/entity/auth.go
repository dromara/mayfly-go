package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type OAuthAccount struct {
	model.DeletedModel

	AccountId uint64 `json:"accountId" gorm:"column:account_id;index:account_id,unique"`
	Identity  string `json:"identity" gorm:"column:identity;index:identity,unique"`

	CreateTime *time.Time `json:"createTime"`
	UpdateTime *time.Time `json:"updateTime"`
}

func (OAuthAccount) TableName() string {
	return "t_oauth_account"
}
