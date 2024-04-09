package entity

import "mayfly-go/pkg/model"

type TagTreeQuery struct {
	model.Model

	Pid           uint64
	Type          TagType `json:"type"`
	Code          string  `json:"code"` // 标识
	Codes         []string
	CodePath      string `json:"codePath"` // 标识路径
	CodePaths     []string
	Name          string `json:"name"` // 名称
	CodePathLike  string // 标识符路径模糊查询
	CodePathLikes []string
}

type TeamQuery struct {
	model.Model

	Name string `json:"name" form:"name"` // 团队名称
}
