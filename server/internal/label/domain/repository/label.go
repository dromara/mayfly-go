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

	// GetByKeyIncludingDeleted 查询标签，含已被逻辑删除的行。
	//
	// idx_label_kv 唯一索引不区分是否已删除，被软删的行仍占着这对 key+value；重复校验只看
	// 未删除数据的话，「删掉同名标签再重新添加」会直接撞数据库唯一键报 500
	GetByKeyIncludingDeleted(labelKey, labelValue string) (*entity.Label, error)

	// ListKeys 获取所有标签键（去重）
	ListKeys(ctx context.Context) ([]string, error)

	// ListValuesByKey 获取指定 key 的所有值
	ListValuesByKey(ctx context.Context, labelKey string) ([]string, error)
}

// LabelBinding 标签绑定仓库接口
type LabelBinding interface {
	base.Repo[*entity.LabelBinding]

	// DeleteByLabelId 根据标签ID删除所有绑定
	DeleteByLabelId(ctx context.Context, labelId uint64) error

	// CountByLabelId 统计标签被绑定的数量（用于删除前的引用校验）。
	//
	// 必须回报错误：统计失败若被当成「0 条绑定」，删除校验会直接放行，留下指向已删标签的孤儿绑定
	CountByLabelId(ctx context.Context, labelId uint64) (int64, error)

	// ListByTarget 根据目标类型+ID查询绑定
	ListByTarget(targetType string, targetId uint64) ([]*entity.LabelBinding, error)

	// ListByTargetIds 批量查询多个目标的绑定（消除 N+1）
	ListByTargetIds(targetType string, targetIds []uint64) ([]*entity.LabelBinding, error)

	// ListLabelIdsByTarget 根据目标类型+ID查询绑定的标签ID列表
	ListLabelIdsByTarget(ctx context.Context, targetType string, targetId uint64) ([]uint64, error)

	// ListDeletedLabelIds 在给定标签中查出「该目标曾绑定过但已被逻辑删除」的标签 id。
	//
	// idx_lb_unique 唯一索引对墓碑行同样生效：解绑再绑同一标签时，若只按未删除数据做差异比较，
	// 新增项会落到 INSERT 上并撞唯一键
	ListDeletedLabelIds(ctx context.Context, targetType string, targetId uint64, labelIds []uint64) ([]uint64, error)
}
