package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/model"
)

type syslogRepo struct{}

var SyslogDao repository.Syslog = &syslogRepo{}

func (m *syslogRepo) GetPageList(condition *entity.Syslog, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

func (m *syslogRepo) Insert(syslog *entity.Syslog) {
	model.Insert(syslog)
}
