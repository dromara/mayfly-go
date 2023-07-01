package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/model"
)

type Syslog interface {
	GetPageList(condition *entity.Syslog, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Insert(log *entity.Syslog)
}
