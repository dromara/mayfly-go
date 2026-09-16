package persistence

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type alertInhibitionRepoImpl struct {
	base.RepoImpl[*entity.AlertInhibition]
}

var _ repository.AlertInhibition = (*alertInhibitionRepoImpl)(nil)

func newAlertInhibitionRepo() repository.AlertInhibition {
	return &alertInhibitionRepoImpl{}
}

func (r *alertInhibitionRepoImpl) GetAlertInhibitionList(condition *entity.AlertInhibitionQuery, orderBy ...string) (*model.PageResult[*entity.AlertInhibition], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("status", condition.Status)

	return r.PageByCond(qd, condition.PageParam)
}

func (r *alertInhibitionRepoImpl) ListEnabled() ([]*entity.AlertInhibition, error) {
	return r.SelectByCond(model.NewCond().Eq("status", entity.AlertInhibitionStatusEnable))
}
