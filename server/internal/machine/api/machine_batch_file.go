package api

import (
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

// MachineBatchFile 多机批量文件分发接口
type MachineBatchFile struct {
	batchFileApp application.MachineBatchFile `inject:"T"`
}

func (mbf *MachineBatchFile) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 分发文件到多台机器，属命令执行类操作，挂 machine:terminal 并留审计
		req.NewPost("batch-file", mbf.Dispatch).Log(req.NewLogSaveI(imsg.LogMachineBatchFile)).RequiredPermissionCode("machine:terminal"),
	}

	return req.NewConfs("machines", reqs[:]...)
}

// batchFileForm 批量文件分发入参：文件先经平台文件服务上传得到 fileKey，再分发到各机
type batchFileForm struct {
	MachineIds []uint64 `json:"machineIds"`
	FileKey    string   `json:"fileKey"`
	RemotePath string   `json:"remotePath"`
}

// Dispatch 把文件服务中的文件分发到多台机器的目标目录，逐台聚合结果
func (mbf *MachineBatchFile) Dispatch(rc *req.Ctx) {
	form := rc.BindJson[batchFileForm]()
	rc.ReqParam = form

	results, err := mbf.batchFileApp.DispatchFile(rc.MetaCtx, form.MachineIds, form.FileKey, form.RemotePath)
	biz.ErrIsNil(err)
	rc.ResData = results
}
