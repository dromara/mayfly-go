package repository

import (
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Redis interface {
	base.Repo[*entity.Redis]

	// 分页获取机器信息列表
	GetRedisList(condition *entity.RedisQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}
