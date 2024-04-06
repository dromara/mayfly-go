package entity

import (
	"mayfly-go/pkg/model"
	"strings"
)

// 标签树
type TagTree struct {
	model.Model

	Pid      uint64 `json:"pid"`
	Type     int8   `json:"type"`     // 类型： -1.普通标签； 其他值则为对应的资源类型
	Code     string `json:"code"`     // 标识编码, 若类型不为-1，则为对应资源编码
	CodePath string `json:"codePath"` // 标识路径
	Name     string `json:"name"`     // 名称
	Remark   string `json:"remark"`   // 备注说明
}

const (
	// 标识路径分隔符
	CodePathSeparator = "/"
)

// GetRootCode 获取根路径信息
func (pt *TagTree) GetRootCode() string {
	return strings.Split(pt.CodePath, CodePathSeparator)[0]
}

// GetParentPath 获取父标签路径, 如CodePath = test/test1/test2/  -> test/test1/
func (pt *TagTree) GetParentPath() string {
	// 去掉末尾的分隔符
	input := strings.TrimRight(pt.CodePath, CodePathSeparator)

	// 查找倒数第二个连字符位置
	lastHyphenIndex := strings.LastIndex(input, CodePathSeparator)
	if lastHyphenIndex == -1 {
		return ""
	}

	// 截取字符串
	return input[:lastHyphenIndex+1]
}

// 标签接口资源，如果要实现资源结构体填充标签信息，则资源结构体需要实现该接口
type ITagResource interface {
	// 获取资源code
	GetCode() string

	// 赋值标签基本信息
	SetTagInfo(rt ResourceTag)
}

// 资源关联的标签信息
type ResourceTag struct {
	TagId   uint64 `json:"tagId" gorm:"-"`
	TagPath string `json:"tagPath" gorm:"-"` // 标签路径
}

func (r *ResourceTag) SetTagInfo(rt ResourceTag) {
	r.TagId = rt.TagId
	r.TagPath = rt.TagPath
}

// 资源标签列表
type ResourceTags struct {
	Tags []ResourceTag `json:"tags" gorm:"-"`
}

func (r *ResourceTags) SetTagInfo(rt ResourceTag) {
	if r.Tags == nil {
		r.Tags = make([]ResourceTag, 0)
	}
	r.Tags = append(r.Tags, rt)
}
