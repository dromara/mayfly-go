package api

import (
	"context"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"time"
)

func (r *Redis) GetSetValue(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	res, err := ri.GetCmdable().SMembers(context.TODO(), key).Result()
	biz.ErrIsNilAppendErr(err, "获取set值失败: %s")
	rc.ResData = res
}

func (r *Redis) SaveSetValue(rc *req.Ctx) {
	keyvalue := req.BindJsonAndValid(rc, new(form.SetValue))

	cmd := r.getRedisConn(rc).GetCmdable()

	key := keyvalue.Key
	// 简单处理->先删除，后新增
	cmd.Del(context.TODO(), key)
	cmd.SAdd(context.TODO(), key, keyvalue.Value...)

	if keyvalue.Timed != -1 {
		cmd.Expire(context.TODO(), key, time.Second*time.Duration(keyvalue.Timed))
	}
}

func (r *Redis) Scard(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)

	total, err := ri.GetCmdable().SCard(context.TODO(), key).Result()
	biz.ErrIsNilAppendErr(err, "scard失败: %s")
	rc.ResData = total
}

func (r *Redis) Sscan(rc *req.Ctx) {
	scan := req.BindJsonAndValid(rc, new(form.ScanForm))

	cmd := r.getRedisConn(rc).GetCmdable()
	keys, cursor, err := cmd.SScan(context.TODO(), scan.Key, scan.Cursor, scan.Match, scan.Count).Result()
	biz.ErrIsNilAppendErr(err, "sscan失败: %s")
	rc.ResData = collx.M{
		"keys":   keys,
		"cursor": cursor,
	}
}

func (r *Redis) Sadd(rc *req.Ctx) {
	option := req.BindJsonAndValid(rc, new(form.SmemberOption))

	cmd := r.getRedisConn(rc).GetCmdable()

	res, err := cmd.SAdd(context.TODO(), option.Key, option.Member).Result()
	biz.ErrIsNilAppendErr(err, "sadd失败: %s")
	rc.ResData = res
}

func (r *Redis) Srem(rc *req.Ctx) {
	option := req.BindJsonAndValid(rc, new(form.SmemberOption))

	cmd := r.getRedisConn(rc).GetCmdable()
	res, err := cmd.SRem(context.TODO(), option.Key, option.Member).Result()
	biz.ErrIsNilAppendErr(err, "srem失败: %s")
	rc.ResData = res
}
