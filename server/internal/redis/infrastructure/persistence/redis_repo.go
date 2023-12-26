package persistence

import (
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type redisRepoImpl struct {
	base.RepoImpl[*entity.Redis]
}

func newRedisRepo() repository.Redis {
	return &redisRepoImpl{base.RepoImpl[*entity.Redis]{M: new(entity.Redis)}}
}

// 分页获取机器信息列表
func (r *redisRepoImpl) GetRedisList(condition *entity.RedisQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.Redis)).
		Like("host", condition.Host).
		In("code", condition.Codes)
	return gormx.PageQuery(qd, pageParam, toEntity)
}
