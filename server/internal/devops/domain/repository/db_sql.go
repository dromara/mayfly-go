package repository

import "mayfly-go/internal/devops/domain/entity"

type DbSql interface {
	DeleteBy(condition *entity.DbSql)
}
