package entity

import "mayfly-go/pkg/model"

// 标签树与团队关联信息
type TagTreeTeam struct {
	model.Model

	TagId   uint64 `json:"tagId"`
	TagPath string `json:"tagPath"`
	TeamId  uint64 `json:"teamId"`
}
