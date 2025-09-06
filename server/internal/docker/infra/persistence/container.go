package persistence

import (
	"mayfly-go/internal/docker/domain/entity"
	"mayfly-go/internal/docker/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type containerRepoImpl struct {
	base.RepoImpl[*entity.Container]
}

func newContainerRepo() repository.Container {
	return &containerRepoImpl{}
}

func (m *containerRepoImpl) GetContainerPage(condition *entity.ContainerQuery, orderBy ...string) (*model.PageResult[*entity.Container], error) {
	qd := model.NewCond().
		Eq("id", condition.Id).
		Like("addr", condition.Addr).
		Like("name", condition.Name).
		In("code", condition.Codes).
		Eq("code", condition.Code)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("addr like ? or name like ? or code like ?", keyword, keyword, keyword)
	}

	return m.PageByCond(qd, condition.PageParam)
}
