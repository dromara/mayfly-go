package entity

import (
	"mayfly-go/pkg/model"
	"strings"
)

// 标签树
type TagTree struct {
	model.Model

	Pid      uint64 `json:"pid"`
	Code     string `json:"code"`     // 标识
	CodePath string `json:"codePath"` // 标识路径
	Name     string `json:"name"`     // 名称
	Remark   string `json:"remark"`   // 备注说明
}

const (
	// 标识路径分隔符
	CodePathSeparator = "/"
)

// 获取根路径信息
func (pt *TagTree) GetRootCode() string {
	return strings.Split(pt.CodePath, CodePathSeparator)[0]
}
