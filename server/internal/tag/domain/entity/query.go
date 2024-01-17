package entity

import "mayfly-go/pkg/model"

type TagTreeQuery struct {
	model.Model

	Pid           uint64
	Code          string `json:"code"`     // 标识
	CodePath      string `json:"codePath"` // 标识路径
	CodePaths     []string
	Name          string `json:"name"` // 名称
	CodePathLike  string // 标识符路径模糊查询
	CodePathLikes []string
}

type TagResourceQuery struct {
	model.Model

	TagPath       string   `json:"string"` // 标签路径
	TagId         uint64   `json:"tagId" form:"tagId"`
	ResourceType  int8     `json:"resourceType" form:"resourceType"` // 资源编码
	ResourceCode  string   `json:"resourceCode" form:"resourceCode"` // 资源编码
	ResourceCodes []string // 资源编码列表

	TagPathLike  string // 标签路径模糊查询
	TagPathLikes []string
}

type TeamQuery struct {
	model.Model

	Name string `json:"name" form:"name"` // 团队名称
}
