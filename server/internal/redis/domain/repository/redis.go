package repository

import (
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/pkg/model"
)

type Redis interface {
	// 分页获取机器信息列表
	GetRedisList(condition *entity.Redis, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Count(condition *entity.Redis) int64

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Redis

	GetRedis(condition *entity.Redis, cols ...string) error

	Insert(redis *entity.Redis)

	Update(redis *entity.Redis)

	Delete(id uint64)
}
