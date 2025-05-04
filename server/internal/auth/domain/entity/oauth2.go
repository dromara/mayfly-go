package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type Oauth2Account struct {
	model.DeletedModel

	AccountId uint64 `json:"accountId" gorm:"not null;column:account_id;index:account_id,unique;comment:账号ID"`
	Identity  string `json:"identity" gorm:"size:64;column:identity;index:idx_identity,unique;comment:身份标识"`

	CreateTime *time.Time `json:"createTime" gorm:"not null;"`
	UpdateTime *time.Time `json:"updateTime" gorm:"not null;"`
}

func (Oauth2Account) TableName() string {
	return "t_oauth2_account"
}
