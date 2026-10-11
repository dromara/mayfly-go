package persistence

import (
	"context"
	"mayfly-go/internal/label/domain/entity"
	"mayfly-go/internal/label/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type labelRepoImpl struct {
	base.RepoImpl[*entity.Label]
}

var _ repository.Label = (*labelRepoImpl)(nil)

func newLabelRepo() repository.Label {
	return &labelRepoImpl{}
}

func (r *labelRepoImpl) GetByKey(labelKey, labelValue string) (*entity.Label, error) {
	var label entity.Label
	err := r.GetByCond(model.NewCond().Eq("label_key", labelKey).Eq("label_value", labelValue).Dest(&label))
	if err != nil {
		if model.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &label, nil
}

// GetByKeyIncludingDeleted 含已逻辑删除的行，用于预判唯一索引冲突
func (r *labelRepoImpl) GetByKeyIncludingDeleted(labelKey, labelValue string) (*entity.Label, error) {
	var label entity.Label
	err := r.GetByCond(model.NewCond().IncludeDeleted().Eq("label_key", labelKey).Eq("label_value", labelValue).Dest(&label))
	if err != nil {
		if model.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &label, nil
}

func (r *labelRepoImpl) ListKeys(ctx context.Context) ([]string, error) {
	var keys []string
	// 表名取自实体而非写死字面量：物理表名由 NamingStrategy 统一派生，写死后改前缀或换方言就会静默失效
	err := r.SelectBySQL(ctx, "SELECT DISTINCT label_key FROM "+(&entity.Label{}).TableName()+" WHERE is_deleted = 0 ORDER BY label_key", &keys)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *labelRepoImpl) ListValuesByKey(ctx context.Context, labelKey string) ([]string, error) {
	var values []string
	err := r.SelectBySQL(ctx, "SELECT DISTINCT label_value FROM "+(&entity.Label{}).TableName()+" WHERE is_deleted = 0 AND label_key = ? ORDER BY label_value", &values, labelKey)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// LabelBinding persistence

type labelBindingRepoImpl struct {
	base.RepoImpl[*entity.LabelBinding]
}

var _ repository.LabelBinding = (*labelBindingRepoImpl)(nil)

func newLabelBindingRepo() repository.LabelBinding {
	return &labelBindingRepoImpl{}
}

func (r *labelBindingRepoImpl) DeleteByLabelId(ctx context.Context, labelId uint64) error {
	return r.DeleteByCond(ctx, model.NewCond().Eq("label_id", labelId))
}

func (r *labelBindingRepoImpl) CountByLabelId(ctx context.Context, labelId uint64) (int64, error) {
	return r.CountByCond(model.NewCond().Eq("label_id", labelId))
}

func (r *labelBindingRepoImpl) ListByTarget(targetType string, targetId uint64) ([]*entity.LabelBinding, error) {
	return r.SelectByCond(model.NewCond().Eq("target_type", targetType).Eq("target_id", targetId))
}

func (r *labelBindingRepoImpl) ListByTargetIds(targetType string, targetIds []uint64) ([]*entity.LabelBinding, error) {
	if len(targetIds) == 0 {
		return nil, nil
	}
	return r.SelectByCond(model.NewCond().Eq("target_type", targetType).In("target_id", targetIds))
}

func (r *labelBindingRepoImpl) ListDeletedLabelIds(ctx context.Context, targetType string, targetId uint64, labelIds []uint64) ([]uint64, error) {
	if len(labelIds) == 0 {
		return nil, nil
	}
	// 方法名承诺「只返回墓碑」，就必须自己写下删除位条件：IncludeDeleted 只是**不过滤**已删除行，
	// 活跃绑定也会一并返回。当前调用方恰好只关心新增集（与删除集互不相交）而侥幸不出错，
	// 但这个条件不该由调用方的不变量来保证
	bindings, err := r.SelectByCond(model.NewCond().IncludeDeleted().
		EqStrict(model.DeletedColumn, model.ModelDeleted).
		Eq("target_type", targetType).
		Eq("target_id", targetId).
		In("label_id", labelIds))
	if err != nil {
		return nil, err
	}
	deleted := make([]uint64, 0, len(bindings))
	for _, binding := range bindings {
		deleted = append(deleted, binding.LabelId)
	}
	return deleted, nil
}

func (r *labelBindingRepoImpl) ListLabelIdsByTarget(ctx context.Context, targetType string, targetId uint64) ([]uint64, error) {
	var ids []uint64
	err := r.SelectBySQL(ctx, "SELECT label_id FROM "+(&entity.LabelBinding{}).TableName()+" WHERE is_deleted = 0 AND target_type = ? AND target_id = ?", &ids, targetType, targetId)
	if err != nil {
		return nil, err
	}
	return ids, nil
}
