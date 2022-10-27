package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/model"
)

type syslogRepoImpl struct{}

func newSyslogRepo() repository.Syslog {
	return new(syslogRepoImpl)
}

func (m *syslogRepoImpl) GetPageList(condition *entity.Syslog, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, condition, toEntity, orderBy...)
}

func (m *syslogRepoImpl) Insert(syslog *entity.Syslog) {
	model.Insert(syslog)
}
