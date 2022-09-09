package repository

import "mayfly-go/internal/db/domain/entity"

type DbSql interface {
	DeleteBy(condition *entity.DbSql)
}
