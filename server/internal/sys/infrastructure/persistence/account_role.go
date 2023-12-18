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
	return &accountRoleRepoImpl{base.RepoImpl[*entity.AccountRole]{M: new(entity.AccountRole)}}
}

func (m *accountRoleRepoImpl) GetPageList(condition *entity.RoleAccountQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQueryWithTableName("t_sys_account_role t").
		Select("t.creator, t.create_time, a.username, a.name accountName, a.status accountStatus, a.id accountId").
		Joins("JOIN t_sys_account a ON t.account_id = a.id AND a.status = 1").
		Eq0("a.is_deleted", model.ModelUndeleted).
		Eq0("t.is_deleted", model.ModelUndeleted).
		RLike("a.username", condition.Username).
		RLike("a.name", condition.Name).
		Eq("t.role_id", condition.RoleId).
		OrderByDesc("t.id")
	return gormx.PageQuery(qd, pageParam, toEntity)
}
