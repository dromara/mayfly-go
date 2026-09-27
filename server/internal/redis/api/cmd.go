package api

import (
	"mayfly-go/internal/pkg/event"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/application/dto"
	"mayfly-go/internal/redis/rdm"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"

	"github.com/spf13/cast"
)

func (r *Redis) RunCmd(rc *req.Ctx) {
	cmdReq, runCmdParam := rc.BindJsonAndCopyTo[form.RunCmdForm, dto.RunCmd]()
	biz.IsTrue(len(cmdReq.Cmd) > 0, "redis cmd cannot be empty")

	// 命令控制台直接透传原始命令，因此权限只能按命令语义判定：高危命令要求删除权限，写命令要求保存权限
	cmdName := cast.ToString(cmdReq.Cmd[0])
	if rdm.IsDangerousCmd(cmdName) {
		r.requireDangerPerm(rc)
	} else if rdm.IsWriteCmd(cmdName) {
		r.requirePerm(rc, permDataSave)
	}

	redisConn := r.getRedisConn(rc)
	biz.ErrIsNilAppendErr(r.tagApp.CanAccess(rc.GetLoginAccount().Id, redisConn.Info.CodePath...), "%s")
	rc.ReqParam = collx.Kvs("redis", redisConn.Info, "cmd", cmdReq.Cmd)

	global.EventBus.Publish(rc.MetaCtx, event.EventTopicResourceOp, redisConn.Info.CodePath[0])

	res, err := r.redisApp.RunCmd(rc.MetaCtx, redisConn, runCmdParam)
	biz.ErrIsNil(err)
	rc.ResData = res
}
