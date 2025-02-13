package entity

import "mayfly-go/pkg/model"

// 与标签树有关联关系的实体
type TagTreeRelate struct {
	model.Model

	TagId      uint64        `json:"tagId" gorm:"not null;index:idx_tag_id;comment:标签树id"` // 标签树id
	RelateId   uint64        `json:"relateId" gorm:"not null;comment:关联的资源id"`             // 关联的资源id
	RelateType TagRelateType `json:"relateType" gorm:"not null;comment:关联类型"`              // 关联的类型
}

type TagRelateType int8

const (
	TagRelateTypeTeam           TagRelateType = 1 // 关联团队
	TagRelateTypeMachineCmd     TagRelateType = 2 // 关联机器命令配置
	TagRelateTypeMachineCronJob TagRelateType = 3 // 关联机器定时任务配置
	TagRelateTypeFlowDef        TagRelateType = 4 // 关联流程定义
)

// 关联标签信息，如果要实现填充关联标签信息，则结构体需要实现该接口
type IRelateTag interface {
	// 获取关联id
	GetRelateId() uint64

	// 赋值标签路径
	SetTagInfo(tag ResourceTag)
}

// 关联的标签信息
type RelateTags struct {
	Tags []ResourceTag `json:"tags" gorm:"-"` // 标签路径
}

func (r *RelateTags) SetTagInfo(rt ResourceTag) {
	if r.Tags == nil {
		r.Tags = make([]ResourceTag, 0)
	}
	r.Tags = append(r.Tags, rt)
}
