package repository

import (
	"context"
	"mayfly-go/internal/label/domain/entity"
	"mayfly-go/pkg/base"
)

// Label 标签注册表仓库接口
type Label interface {
	base.Repo[*entity.Label]

	// GetByKey 根据 key+value 查询标签
	GetByKey(labelKey, labelValue string) (*entity.Label, error)

	// ListKeys 获取所有标签键（去重）
	ListKeys() ([]string, error)

	// ListValuesByKey 获取指定 key 的所有值
	ListValuesByKey(labelKey string) ([]string, error)
}

// LabelBinding 标签绑定仓库接口
type LabelBinding interface {
	base.Repo[*entity.LabelBinding]

	// DeleteByLabelId 根据标签ID删除所有绑定
	DeleteByLabelId(ctx context.Context, labelId uint64) error

	// CountByLabelId 统计标签被绑定的数量（用于删除前的引用校验）
	CountByLabelId(labelId uint64) int64

	// ListByTarget 根据目标类型+ID查询绑定
	ListByTarget(targetType string, targetId uint64) ([]*entity.LabelBinding, error)

	// ListByTargetIds 批量查询多个目标的绑定（消除 N+1）
	ListByTargetIds(targetType string, targetIds []uint64) ([]*entity.LabelBinding, error)

	// ListLabelIdsByTarget 根据目标类型+ID查询绑定的标签ID列表
	ListLabelIdsByTarget(targetType string, targetId uint64) ([]uint64, error)
}
