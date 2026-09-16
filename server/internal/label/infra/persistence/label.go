package persistence

import (
	"context"
	"errors"
	"mayfly-go/internal/label/domain/entity"
	"mayfly-go/internal/label/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"

	"gorm.io/gorm"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &label, nil
}

func (r *labelRepoImpl) ListKeys() ([]string, error) {
	var keys []string
	err := r.SelectBySql("SELECT DISTINCT label_key FROM t_label WHERE is_deleted = 0 ORDER BY label_key", &keys)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *labelRepoImpl) ListValuesByKey(labelKey string) ([]string, error) {
	var values []string
	err := r.SelectBySql("SELECT DISTINCT label_value FROM t_label WHERE is_deleted = 0 AND label_key = ? ORDER BY label_value", &values, labelKey)
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

func (r *labelBindingRepoImpl) CountByLabelId(labelId uint64) int64 {
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

func (r *labelBindingRepoImpl) ListLabelIdsByTarget(targetType string, targetId uint64) ([]uint64, error) {
	var ids []uint64
	err := r.SelectBySql("SELECT label_id FROM t_label_binding WHERE is_deleted = 0 AND target_type = ? AND target_id = ?", &ids, targetType, targetId)
	if err != nil {
		return nil, err
	}
	return ids, nil
}
