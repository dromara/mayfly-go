package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type syslogRepoImpl struct{}

func newSyslogRepo() repository.Syslog {
	return new(syslogRepoImpl)
}

func (m *syslogRepoImpl) GetPageList(condition *entity.Syslog, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (m *syslogRepoImpl) Insert(syslog *entity.Syslog) {
	gormx.Insert(syslog)
}
