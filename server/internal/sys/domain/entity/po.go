package entity

import "time"

type AccountRolePO struct {
	RoleId        uint64        `json:"roleId"`
	RoleName      string        `json:"roleName"`
	Code          string        `json:"code"`
	Status        int           `json:"status"`
	AccountId     uint64        `json:"accountId" gorm:"column:accountId"`
	AccountName   string        `json:"accountName" gorm:"column:accountName"`
	Username      string        `json:"username"`
	AccountStatus AccountStatus `json:"accountStatus" gorm:"column:accountStatus"`
	CreateTime    *time.Time    `json:"createTime"`
	Creator       string        `json:"creator"`
}
