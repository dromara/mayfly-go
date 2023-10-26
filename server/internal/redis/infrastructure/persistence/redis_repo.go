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
		In("tag_id", condition.TagIds).
		RLike("tag_path", condition.TagPath).
		OrderByAsc("tag_path")
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (r *redisRepoImpl) Count(condition *entity.RedisQuery) int64 {
	where := make(map[string]any)
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}

	return gormx.CountByCond(new(entity.Redis), where)
}
