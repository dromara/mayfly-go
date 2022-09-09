package entity

import "mayfly-go/pkg/model"

// 项目成员，用于对项目下组件的访问控制
type ProjectMember struct {
	model.Model
	AccountId uint64 `json:"accountId"` // 账号
	Username  string `json:"username"`  // 账号用户名
	ProjectId uint64 `json:"projectId"` // 项目id
}
