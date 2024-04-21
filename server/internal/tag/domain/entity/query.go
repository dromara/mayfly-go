package entity

import "mayfly-go/pkg/model"

type TagTreeQuery struct {
	model.Model

	Type          TagType `json:"type"`
	Codes         []string
	CodePaths     []string // 标识路径
	Name          string   `json:"name"` // 名称
	CodePathLike  string   // 标识符路径模糊查询
	CodePathLikes []string
}

type TeamQuery struct {
	model.Model

	Name string `json:"name" form:"name"` // 团队名称
}
