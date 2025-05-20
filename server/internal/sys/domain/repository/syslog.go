package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Syslog interface {
	base.Repo[*entity.SysLog]

	GetPageList(condition *entity.SysLogQuery, orderBy ...string) (*model.PageResult[*entity.SysLog], error)
}
