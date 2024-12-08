package dto

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

// 保存资源标签参数
type SaveResourceTag struct {
	ParentTagCodePaths []string // 关联标签，空数组则为删除该资源绑定的标签

	ResourceTag *ResourceTag // 资源标签信息
}

type ResourceTag struct {
	Code string
	Type entity.TagType
	Name string

	Children []*ResourceTag // 子资源标签
}

type RelateTagsByCodeAndType struct {
	ParentTagCode string         // 父标签编号
	ParentTagType entity.TagType // 父标签类型

	Tags []*ResourceTag // 要关联的标签数组
}

type DelResourceTag struct {
	Id           uint64
	ResourceCode string
	ResourceType entity.TagType

	// 要删除的子节点类型，若存在值，则为删除资源标签下的指定类型的子标签
	ChildType entity.TagType
}

type SimpleTagTree struct {
	Id       uint64         `json:"id"`
	Type     entity.TagType `json:"type"`     // 类型： -1.普通标签； 其他值则为对应的资源类型
	Code     string         `json:"code"`     // 标识编码, 若类型不为-1，则为对应资源编码
	CodePath string         `json:"codePath"` // 标识路径，tag1/tag2/tagType1|tagCode/tagType2|yyycode/，非普通标签类型段含有标签类型
	Name     string         `json:"name"`     // 名称
	Remark   string         `json:"remark"`
	Root     bool           `json:"-" gorm:"-"`
}

func (pt *SimpleTagTree) IsRoot() bool {
	// 去除路径两端可能存在的斜杠
	path := strings.Trim(string(pt.CodePath), "/")
	return len(strings.Split(path, "/")) == 1
}

// GetParentPath 获取父标签路径, 如CodePath = test/test1/test2/  -> test/test1/
func (pt *SimpleTagTree) GetParentPath() string {
	return string(entity.CodePath(pt.CodePath).GetParent(0))
}

type SimpleTagTrees []*SimpleTagTree

// GetCodes 获取code数组
func (ts SimpleTagTrees) GetCodes() []string {
	// resouce code去重
	code2Resource := collx.ArrayToMap[*SimpleTagTree, string](ts, func(val *SimpleTagTree) string {
		return val.Code
	})

	return collx.MapKeys(code2Resource)
}

// GetCodePaths 获取codePath数组
func (ts SimpleTagTrees) GetCodePaths() []string {
	// codepath去重
	codepath2Resource := collx.ArrayToMap[*SimpleTagTree, string](ts, func(val *SimpleTagTree) string {
		return val.CodePath
	})

	return collx.MapKeys(codepath2Resource)
}
