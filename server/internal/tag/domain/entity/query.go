package entity

import (
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type TypePath string

// ToTagTypes 转为路径对应的TagType数组
func (t TypePath) ToTagTypes() []TagType {
	return collx.ArrayMap(strings.Split(string(t), CodePathSeparator), func(val string) TagType { return TagType(cast.ToInt8(val)) })
}

// NewTypePaths
func NewTypePaths(types ...TagType) TypePath {
	return TypePath(strings.Join(collx.ArrayMap[TagType, string](types, func(val TagType) string { return cast.ToString(int8(val)) }), CodePathSeparator))
}

type TagTreeQuery struct {
	model.PageParam
	model.Model

	Types         []TagType
	TypePaths     []TypePath // 类型路径。如 machineType/authcertType，即获取机器下的授权凭证
	Codes         []string
	CodePaths     []string // 标识路径
	Name          string   `json:"name"` // 名称
	CodePathLikes []string

	GetAllChildren bool // 是否获取所有子节点
}

type TeamQuery struct {
	model.PageParam
	model.Model

	Name string `json:"name" form:"name"` // 团队名称
}
