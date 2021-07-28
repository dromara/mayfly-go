package repository

import "mayfly-go/server/devops/domain/entity"

type DbSql interface {
	DeleteBy(condition *entity.DbSql)
}
