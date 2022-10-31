package persistence

import (
	"fmt"
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"strings"
)

type redisRepoImpl struct{}

func newRedisRepo() repository.Redis {
	return new(redisRepoImpl)
}

// 分页获取机器信息列表
func (r *redisRepoImpl) GetRedisList(condition *entity.RedisQuery, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	sql := "SELECT d.* FROM t_redis d WHERE 1=1  "
	values := make([]interface{}, 0)
	if condition.Host != "" {
		sql = sql + " AND d.host LIKE ?"
		values = append(values, "%"+condition.Host+"%")
	}
	if len(condition.TagIds) > 0 {
		sql = sql + " AND d.tag_id IN " + fmt.Sprintf("(%s)", strings.Join(utils.NumberArr2StrArr(condition.TagIds), ","))
	}
	if condition.TagPathLike != "" {
		sql = sql + " AND d.tag_path LIKE ?"
		values = append(values, condition.TagPathLike+"%")
	}
	sql = sql + " ORDER BY d.tag_path"
	return model.GetPageBySql(sql, pageParam, toEntity, values...)
}

func (r *redisRepoImpl) Count(condition *entity.RedisQuery) int64 {
	where := make(map[string]interface{})
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}
	if condition.TagId != 0 {
		where["tag_id"] = condition.TagId
	}

	return model.CountByMap(new(entity.Redis), where)
}

// 根据id获取
func (r *redisRepoImpl) GetById(id uint64, cols ...string) *entity.Redis {
	rd := new(entity.Redis)
	if err := model.GetById(rd, id, cols...); err != nil {
		return nil
	}
	return rd
}

func (r *redisRepoImpl) GetRedis(condition *entity.Redis, cols ...string) error {
	return model.GetBy(condition, cols...)
}

func (r *redisRepoImpl) Insert(redis *entity.Redis) {
	biz.ErrIsNilAppendErr(model.Insert(redis), "新增失败: %s")
}

func (r *redisRepoImpl) Update(redis *entity.Redis) {
	biz.ErrIsNilAppendErr(model.UpdateById(redis), "更新失败: %s")
}

func (r *redisRepoImpl) Delete(id uint64) {
	biz.ErrIsNilAppendErr(model.DeleteById(new(entity.Redis), id), "删除失败: %s")
}
