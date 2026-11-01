package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

type DbColumnAcl interface {
	base.Repo[*entity.DbColumnAcl]

	// ListByDb 取某库配置的全部列级规则（按表名、列名升序）
	ListByDb(dbId uint64) ([]*entity.DbColumnAcl, error)
}
