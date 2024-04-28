package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type roleRepoImpl struct {
	base.RepoImpl[*entity.Role]
}

func newRoleRepo() repository.Role {
	return &roleRepoImpl{base.RepoImpl[*entity.Role]{M: new(entity.Role)}}
}

func (m *roleRepoImpl) GetPageList(condition *entity.RoleQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Like("code", condition.Code).
		In("id", condition.Ids).
		NotIn("id", condition.NotIds).
		OrderBy(orderBy...)
	return m.PageByCond(qd, pageParam, toEntity)
}

func (m *roleRepoImpl) ListByQuery(condition *entity.RoleQuery) ([]*entity.Role, error) {
	var res []*entity.Role
	qd := model.NewCond().
		Like("name", condition.Name).
		Like("code", condition.Code).
		In("id", condition.Ids).
		NotIn("id", condition.NotIds).
		OrderByDesc("id")
	err := m.SelectByCond(qd, &res)
	return res, err
}
