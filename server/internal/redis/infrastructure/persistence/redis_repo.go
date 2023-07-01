package persistence

import (
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type redisRepoImpl struct{}

func newRedisRepo() repository.Redis {
	return new(redisRepoImpl)
}

// 分页获取机器信息列表
func (r *redisRepoImpl) GetRedisList(condition *entity.RedisQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(new(entity.Redis)).
		Like("host", condition.Host).
		In("tag_id", condition.TagIds).
		RLike("tag_path", condition.TagPathLike).
		OrderByAsc("tag_path")
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (r *redisRepoImpl) Count(condition *entity.RedisQuery) int64 {
	where := make(map[string]any)
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}
	if condition.TagId != 0 {
		where["tag_id"] = condition.TagId
	}

	return gormx.CountByCond(new(entity.Redis), where)
}

// 根据id获取
func (r *redisRepoImpl) GetById(id uint64, cols ...string) *entity.Redis {
	rd := new(entity.Redis)
	if err := gormx.GetById(rd, id, cols...); err != nil {
		return nil
	}
	return rd
}

func (r *redisRepoImpl) GetRedis(condition *entity.Redis, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

func (r *redisRepoImpl) Insert(redis *entity.Redis) {
	biz.ErrIsNilAppendErr(gormx.Insert(redis), "新增失败: %s")
}

func (r *redisRepoImpl) Update(redis *entity.Redis) {
	biz.ErrIsNilAppendErr(gormx.UpdateById(redis), "更新失败: %s")
}

func (r *redisRepoImpl) Delete(id uint64) {
	biz.ErrIsNilAppendErr(gormx.DeleteById(new(entity.Redis), id), "删除失败: %s")
}
