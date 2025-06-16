package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/may-fly/cast"
)

type MachineScript struct {
	machineScriptApp application.MachineScript `inject:"T"`
	machineApp       application.Machine       `inject:"T"`
	tagApp           tagapp.TagTree            `inject:"T"`
}

func (ms *MachineScript) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取指定机器脚本列表
		req.NewGet(":machineId/scripts", ms.MachineScripts),

		req.NewGet("/scripts/categorys", ms.MachineScriptCategorys),

		req.NewPost(":machineId/scripts", ms.SaveMachineScript).Log(req.NewLogSave("机器-保存脚本")).RequiredPermissionCode("machine:script:save"),

		req.NewDelete(":machineId/scripts/:scriptId", ms.DeleteMachineScript).Log(req.NewLogSave("机器-删除脚本")).RequiredPermissionCode("machine:script:del"),

		req.NewGet("scripts/:scriptId/:ac/run", ms.RunMachineScript).Log(req.NewLogSave("机器-执行脚本")).RequiredPermissionCode("machine:script:run"),
	}

	return req.NewConfs("machines", reqs[:]...)
}

func (m *MachineScript) MachineScripts(rc *req.Ctx) {
	condition := &entity.MachineScript{MachineId: GetMachineId(rc), Category: rc.Query("category")}
	res, err := m.machineScriptApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = model.PageResultConv[*entity.MachineScript, *vo.MachineScriptVO](res)
}

func (m *MachineScript) MachineScriptCategorys(rc *req.Ctx) {
	res, err := m.machineScriptApp.GetScriptCategorys(rc.MetaCtx)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *MachineScript) SaveMachineScript(rc *req.Ctx) {
	form, machineScript := req.BindJsonAndCopyTo[*form.MachineScriptForm, *entity.MachineScript](rc)

	rc.ReqParam = form
	biz.ErrIsNil(m.machineScriptApp.Save(rc.MetaCtx, machineScript))
}

func (m *MachineScript) DeleteMachineScript(rc *req.Ctx) {
	idsStr := rc.PathParam("scriptId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		m.machineScriptApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

func (m *MachineScript) RunMachineScript(rc *req.Ctx) {
	scriptId := GetMachineScriptId(rc)
	ac := GetMachineAc(rc)
	ms, err := m.machineScriptApp.GetById(scriptId, "MachineId", "Name", "Script")
	biz.ErrIsNil(err, "script not found")

	script := ms.Script
	// 如果有脚本参数，则用脚本参数替换脚本中的模板占位符参数
	if params := rc.Query("params"); params != "" {
		p, err := jsonx.ToMap(params)
		biz.ErrIsNil(err)
		script, err = stringx.TemplateParse(ms.Script, p)
		biz.ErrIsNilAppendErr(err, "failed to parse the script template parameter: %s")
	}
	cli, err := m.machineApp.GetCliByAc(rc.MetaCtx, ac)
	biz.ErrIsNilAppendErr(err, "connection error: %s")

	biz.ErrIsNilAppendErr(m.tagApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.CodePath...), "%s")

	res, err := cli.Run(script)
	// 记录请求参数
	rc.ReqParam = collx.Kvs("machine", cli.Info, "scriptId", scriptId, "name", ms.Name)
	if res == "" {
		biz.ErrIsNilAppendErr(err, "failed to execute: %s")
	}
	rc.ResData = res
}

func GetMachineScriptId(rc *req.Ctx) uint64 {
	scriptId := rc.PathParamInt("scriptId")
	biz.IsTrue(scriptId > 0, "scriptId error")
	return uint64(scriptId)
}
