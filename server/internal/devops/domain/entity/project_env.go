package entity

import "mayfly-go/pkg/model"

// 项目环境
type ProjectEnv struct {
	model.Model
	Name      string `json:"name"`      // 环境名
	ProjectId uint64 `json:"projectId"` // 项目id
	Remark    string `json:"remark"`    // 备注说明
}
