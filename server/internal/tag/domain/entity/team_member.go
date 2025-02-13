package entity

import "mayfly-go/pkg/model"

// 团队成员关联信息
type TeamMember struct {
	model.Model

	TeamId    uint64 `json:"teamId" gorm:"not null;"`
	AccountId uint64 `json:"accountId" gorm:"not null;"`
	Username  string `json:"username" gorm:"size:50;not null;"`
}
