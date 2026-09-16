package persistence

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
	"time"
)

type alertSilenceRepoImpl struct {
	base.RepoImpl[*entity.AlertSilence]
}

var _ repository.AlertSilence = (*alertSilenceRepoImpl)(nil)

func newAlertSilenceRepo() repository.AlertSilence {
	return &alertSilenceRepoImpl{}
}

func (r *alertSilenceRepoImpl) GetAlertSilenceList(condition *entity.AlertSilenceQuery, orderBy ...string) (*model.PageResult[*entity.AlertSilence], error) {
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

func (r *alertSilenceRepoImpl) ListActive() ([]*entity.AlertSilence, error) {
	now := time.Now()
	return r.SelectByCond(model.NewCond().
		Eq("status", entity.AlertSilenceStatusEnable).
		And("start_time <= ?", now).
		And("end_time >= ?", now))
}
