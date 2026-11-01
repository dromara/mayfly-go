package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

type DbTableAcl interface {
	base.Repo[*entity.DbTableAcl]

	// ListByDb 取某库配置的全部表访问规则（按表名升序，便于界面稳定展示）
	ListByDb(dbId uint64) ([]*entity.DbTableAcl, error)
}
