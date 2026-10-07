package api

import (
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"

	"github.com/spf13/cast"
)

// MachineDisk 机器磁盘占用分析接口
type MachineDisk struct {
	diskApp application.MachineDisk `inject:"T"`
}

func (md *MachineDisk) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 触发远端 du，属命令执行类，挂 machine:terminal 并留审计
		req.NewPost(":machineId/disk-analyze", md.Analyze).Log(req.NewLogSaveI(imsg.LogMachineDiskAnalyze)).RequiredPermissionCode("machine:terminal"),
	}

	return req.NewConfs("machines", reqs[:]...)
}

// diskAnalyzeForm 磁盘分析入参
type diskAnalyzeForm struct {
	Path  string `json:"path"`
	Depth int    `json:"depth"`
}

// Analyze 分析指定机器某路径下的目录占用
func (md *MachineDisk) Analyze(rc *req.Ctx) {
	machineId := cast.ToUint64(rc.PathParam("machineId"))

	form := rc.BindJson[diskAnalyzeForm]()
	rc.ReqParam = form

	res, err := md.diskApp.Analyze(rc.MetaCtx, machineId, form.Path, form.Depth)
	biz.ErrIsNil(err)
	rc.ResData = res
}
