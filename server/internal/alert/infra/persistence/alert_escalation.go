package persistence

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type alertEscalationRepoImpl struct {
	base.RepoImpl[*entity.AlertEscalation]
}

var _ repository.AlertEscalation = (*alertEscalationRepoImpl)(nil)

func newAlertEscalationRepo() repository.AlertEscalation {
	return &alertEscalationRepoImpl{}
}

func (r *alertEscalationRepoImpl) GetAlertEscalationList(condition *entity.AlertEscalationQuery, orderBy ...string) (*model.PageResult[*entity.AlertEscalation], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("status", condition.Status)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("name like ? or remark like ?", keyword, keyword)
	}
	return r.PageByCond(qd, condition.PageParam)
}

func (r *alertEscalationRepoImpl) ListEnabled() ([]*entity.AlertEscalation, error) {
	return r.SelectByCond(model.NewCond().Eq("status", entity.AlertEscalationStatusEnable))
}
