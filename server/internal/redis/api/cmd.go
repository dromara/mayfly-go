package api

import (
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

func (r *Redis) RunCmd(rc *req.Ctx) {
	var cmdReq form.RunCmdForm
	runCmdParam := req.BindJsonAndCopyTo(rc, &cmdReq, new(application.RunCmdParam))
	biz.IsTrue(len(cmdReq.Cmd) > 0, "redis命令不能为空")
	rc.ReqParam = cmdReq

	res, err := r.RedisApp.RunCmd(rc.MetaCtx, r.getRedisConn(rc), runCmdParam)
	biz.ErrIsNil(err)
	rc.ResData = res
}
