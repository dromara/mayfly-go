package api

import (
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

func (r *Redis) RunCmd(rc *req.Ctx) {
	var cmdReq form.RunCmdForm
	runCmdParam := req.BindJsonAndCopyTo(rc, &cmdReq, new(application.RunCmdParam))
	biz.IsTrue(len(cmdReq.Cmd) > 0, "redis命令不能为空")

	redisConn := r.getRedisConn(rc)
	rc.ReqParam = collx.Kvs("redis", redisConn.Info, "cmd", cmdReq.Cmd)

	res, err := r.RedisApp.RunCmd(rc.MetaCtx, redisConn, runCmdParam)
	biz.ErrIsNil(err)
	rc.ResData = res
}
