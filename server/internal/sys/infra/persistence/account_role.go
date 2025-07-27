package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type accountRoleRepoImpl struct {
	base.RepoImpl[*entity.AccountRole]
}

func newAccountRoleRepo() repository.AccountRole {
	return &accountRoleRepoImpl{}
}

func (m *accountRoleRepoImpl) GetPageList(condition *entity.RoleAccountQuery, orderBy ...string) (*model.PageResult[*entity.AccountRolePO], error) {
	qd := gormx.NewQueryWithTableName("t_sys_account_role t").
		Joins("JOIN t_sys_account a ON t.account_id = a.id AND a.status = 1").
		WithCond(model.NewCond().Columns("t.creator, t.create_time, a.username, a.name accountName, a.status accountStatus, a.id accountId").
			Eq0("a.is_deleted", model.ModelUndeleted).
			Eq0("t.is_deleted", model.ModelUndeleted).
			RLike("a.username", condition.Username).
			RLike("a.name", condition.Name).
			Eq("t.role_id", condition.RoleId).
			OrderByDesc("t.id"))

	var res []*entity.AccountRolePO
	return gormx.PageQuery(qd, condition.PageParam, res)
}
