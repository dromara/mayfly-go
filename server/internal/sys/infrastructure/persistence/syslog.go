package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type syslogRepoImpl struct {
	base.RepoImpl[*entity.SysLog]
}

func newSyslogRepo() repository.Syslog {
	return &syslogRepoImpl{base.RepoImpl[*entity.SysLog]{M: new(entity.SysLog)}}
}

func (m *syslogRepoImpl) GetPageList(condition *entity.SysLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.SysLog)).Like("description", condition.Description).
		Eq("creator_id", condition.CreatorId).Eq("type", condition.Type).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}
