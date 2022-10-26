package entity

import "mayfly-go/pkg/model"

// 团队成员关联信息
type TeamMember struct {
	model.Model

	TeamId    uint64 `json:"teamId"`
	AccountId uint64 `json:"accountId"`
	Username  string `json:"username"`
}
