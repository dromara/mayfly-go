package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/model"
)

type Syslog interface {
	GetPageList(condition *entity.Syslog, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Insert(log *entity.Syslog)
}
