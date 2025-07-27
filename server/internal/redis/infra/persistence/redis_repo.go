package persistence

import (
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type redisRepoImpl struct {
	base.RepoImpl[*entity.Redis]
}

func newRedisRepo() repository.Redis {
	return &redisRepoImpl{}
}

// 分页获取redis信息列表
func (r *redisRepoImpl) GetRedisList(condition *entity.RedisQuery, orderBy ...string) (*model.PageResult[*entity.Redis], error) {
	qd := model.NewCond().
		Eq("id", condition.Id).
		Like("host", condition.Host).
		Eq("code", condition.Code).
		In("code", condition.Codes)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("host like ? or name like ? or code like ?", keyword, keyword, keyword)
	}

	return r.PageByCond(qd, condition.PageParam)
}
