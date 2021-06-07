package repository

import (
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
)

type Db interface {
	// 分页获取机器信息列表
	GetDbList(condition *entity.Db, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult

	// 根据条件获取账号信息
	GetDb(condition *entity.Db, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Db
}
