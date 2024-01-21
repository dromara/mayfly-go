package api

import (
	"context"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

func (r *Redis) GetListValue(rc *req.Ctx) {
	ri, key := r.checkKeyAndGetRedisConn(rc)
	ctx := context.TODO()
	cmdable := ri.GetCmdable()

	len, err := cmdable.LLen(ctx, key).Result()
	biz.ErrIsNilAppendErr(err, "获取list长度失败: %s")

	g := rc.GinCtx
	start := ginx.QueryInt(g, "start", 0)
	stop := ginx.QueryInt(g, "stop", 10)
	res, err := cmdable.LRange(ctx, key, int64(start), int64(stop)).Result()
	biz.ErrIsNilAppendErr(err, "获取list值失败: %s")

	rc.ResData = collx.M{
		"len":  len,
		"list": res,
	}
}

func (r *Redis) Lrem(rc *req.Ctx) {
	g := rc.GinCtx
	option := new(form.LRemOption)
	ginx.BindJsonAndValid(g, option)

	cmd := r.getRedisConn(rc).GetCmdable()
	res, err := cmd.LRem(context.TODO(), option.Key, int64(option.Count), option.Member).Result()
	biz.ErrIsNilAppendErr(err, "lrem失败: %s")
	rc.ResData = res
}

func (r *Redis) SaveListValue(rc *req.Ctx) {
	g := rc.GinCtx
	listValue := new(form.ListValue)
	ginx.BindJsonAndValid(g, listValue)

	cmd := r.getRedisConn(rc).GetCmdable()

	key := listValue.Key
	ctx := context.TODO()
	for _, v := range listValue.Value {
		cmd.RPush(ctx, key, v)
	}
}

func (r *Redis) Lset(rc *req.Ctx) {
	g := rc.GinCtx
	listSetValue := new(form.ListSetValue)
	ginx.BindJsonAndValid(g, listSetValue)

	ri := r.getRedisConn(rc)

	_, err := ri.GetCmdable().LSet(context.TODO(), listSetValue.Key, listSetValue.Index, listSetValue.Value).Result()
	biz.ErrIsNilAppendErr(err, "list set失败: %s")
}
