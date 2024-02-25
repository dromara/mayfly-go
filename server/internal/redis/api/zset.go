package api

import (
	"context"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"github.com/redis/go-redis/v9"
)

func (r *Redis) ZCard(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)

	total, err := ri.GetCmdable().ZCard(context.TODO(), key).Result()
	biz.ErrIsNilAppendErr(err, "zcard失败: %s")
	rc.ResData = total
}

func (r *Redis) ZScan(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)

	cursor := uint64(rc.QueryIntDefault("cursor", 0))
	match := rc.QueryDefault("match", "*")
	count := rc.QueryIntDefault("count", 50)

	keys, cursor, err := ri.GetCmdable().ZScan(context.TODO(), key, cursor, match, int64(count)).Result()
	biz.ErrIsNilAppendErr(err, "sscan失败: %s")
	rc.ResData = collx.M{
		"keys":   keys,
		"cursor": cursor,
	}
}

func (r *Redis) ZRevRange(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	start := rc.QueryIntDefault("start", 0)
	stop := rc.QueryIntDefault("stop", 50)

	res, err := ri.GetCmdable().ZRevRangeWithScores(context.TODO(), key, int64(start), int64(stop)).Result()
	biz.ErrIsNilAppendErr(err, "ZRevRange失败: %s")
	rc.ResData = res
}

func (r *Redis) ZRem(rc *req.Ctx) {
	option := req.BindJsonAndValid(rc, new(form.SmemberOption))

	cmd := r.getRedisConn(rc).GetCmdable()
	res, err := cmd.ZRem(context.TODO(), option.Key, option.Member).Result()
	biz.ErrIsNilAppendErr(err, "zrem失败: %s")
	rc.ResData = res
}

func (r *Redis) ZAdd(rc *req.Ctx) {
	option := req.BindJsonAndValid(rc, new(form.ZAddOption))

	cmd := r.getRedisConn(rc).GetCmdable()
	zm := redis.Z{
		Score:  option.Score,
		Member: option.Member,
	}
	res, err := cmd.ZAdd(context.TODO(), option.Key, zm).Result()
	biz.ErrIsNilAppendErr(err, "zadd失败: %s")
	rc.ResData = res
}
