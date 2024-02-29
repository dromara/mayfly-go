package entity

import (
	"mayfly-go/pkg/model"
)

// 标签资源关联
type TagResource struct {
	model.Model

	TagId        uint64 `json:"tagId"`
	TagPath      string `json:"tagPath"`      // 标签路径
	ResourceCode string `json:"resourceCode"` // 资源标识
	ResourceType int8   `json:"resourceType"` // 资源类型
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
