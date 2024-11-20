package api

import (
	"mayfly-go/internal/event"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/application/dto"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

func (r *Redis) RunCmd(rc *req.Ctx) {
	var cmdReq form.RunCmdForm
	runCmdParam := req.BindJsonAndCopyTo(rc, &cmdReq, new(dto.RunCmd))
	biz.IsTrue(len(cmdReq.Cmd) > 0, "redis cmd cannot be empty")

	redisConn := r.getRedisConn(rc)
	biz.ErrIsNilAppendErr(r.TagApp.CanAccess(rc.GetLoginAccount().Id, redisConn.Info.CodePath...), "%s")
	rc.ReqParam = collx.Kvs("redis", redisConn.Info, "cmd", cmdReq.Cmd)

	global.EventBus.Publish(rc.MetaCtx, event.EventTopicResourceOp, redisConn.Info.CodePath[0])

	res, err := r.RedisApp.RunCmd(rc.MetaCtx, redisConn, runCmdParam)
	biz.ErrIsNil(err)
	rc.ResData = res
}
