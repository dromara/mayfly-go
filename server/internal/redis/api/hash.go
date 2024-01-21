package api

import (
	"context"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"time"
)

func (r *Redis) Hscan(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	g := rc.GinCtx
	count := ginx.QueryInt(g, "count", 10)
	match := g.Query("match")
	cursor := ginx.QueryInt(g, "cursor", 0)
	contextTodo := context.TODO()

	cmdable := ri.GetCmdable()
	keys, nextCursor, err := cmdable.HScan(contextTodo, key, uint64(cursor), match, int64(count)).Result()
	biz.ErrIsNilAppendErr(err, "hcan err: %s")
	keySize, err := cmdable.HLen(contextTodo, key).Result()
	biz.ErrIsNilAppendErr(err, "hlen err: %s")

	rc.ResData = collx.M{
		"keys":    keys,
		"cursor":  nextCursor,
		"keySize": keySize,
	}
}

func (r *Redis) Hdel(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	field := rc.GinCtx.Query("field")

	rc.ReqParam = collx.Kvs("redis", ri.Info, "key", key, "field", field)
	delRes, err := ri.GetCmdable().HDel(context.TODO(), key, field).Result()
	biz.ErrIsNilAppendErr(err, "hdel err: %s")
	rc.ResData = delRes
}

func (r *Redis) Hget(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	field := rc.GinCtx.Query("field")

	res, err := ri.GetCmdable().HGet(context.TODO(), key, field).Result()
	biz.ErrIsNilAppendErr(err, "hget err: %s")
	rc.ResData = res
}

func (r *Redis) Hset(rc *req.Ctx) {
	g := rc.GinCtx
	hashValue := new(form.HashValue)
	ginx.BindJsonAndValid(g, hashValue)

	hv := hashValue.Value[0]
	ri := r.getRedisConn(rc)
	rc.ReqParam = collx.Kvs("redis", ri.Info, "hash", hv)

	res, err := ri.GetCmdable().HSet(context.TODO(), hashValue.Key, hv["field"].(string), hv["value"]).Result()
	biz.ErrIsNilAppendErr(err, "hset失败: %s")
	rc.ResData = res
}

func (r *Redis) SaveHashValue(rc *req.Ctx) {
	g := rc.GinCtx
	hashValue := new(form.HashValue)
	ginx.BindJsonAndValid(g, hashValue)

	ri := r.getRedisConn(rc)
	rc.ReqParam = collx.Kvs("redis", ri.Info, "hash", hashValue)
	cmd := ri.GetCmdable()

	key := hashValue.Key
	contextTodo := context.TODO()
	for _, v := range hashValue.Value {
		res := cmd.HSet(contextTodo, key, v["field"].(string), v["value"])
		biz.ErrIsNilAppendErr(res.Err(), "保存hash值失败: %s")
	}
	if hashValue.Timed != 0 && hashValue.Timed != -1 {
		cmd.Expire(context.TODO(), key, time.Second*time.Duration(hashValue.Timed))
	}
}
