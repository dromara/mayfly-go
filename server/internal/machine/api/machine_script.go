package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/may-fly/cast"
)

type MachineScript struct {
	MachineScriptApp application.MachineScript `inject:""`
	MachineApp       application.Machine       `inject:""`
	TagApp           tagapp.TagTree            `inject:"TagTreeApp"`
}

func (m *MachineScript) MachineScripts(rc *req.Ctx) {
	condition := &entity.MachineScript{MachineId: GetMachineId(rc)}
	res, err := m.MachineScriptApp.GetPageList(condition, rc.GetPageParam(), new([]vo.MachineScriptVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *MachineScript) SaveMachineScript(rc *req.Ctx) {
	form := new(form.MachineScriptForm)
	machineScript := req.BindJsonAndCopyTo(rc, form, new(entity.MachineScript))

	rc.ReqParam = form
	biz.ErrIsNil(m.MachineScriptApp.Save(rc.MetaCtx, machineScript))
}

func (m *MachineScript) DeleteMachineScript(rc *req.Ctx) {
	idsStr := rc.PathParam("scriptId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		m.MachineScriptApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

func (m *MachineScript) RunMachineScript(rc *req.Ctx) {
	scriptId := GetMachineScriptId(rc)
	ac := GetMachineAc(rc)
	ms, err := m.MachineScriptApp.GetById(scriptId, "MachineId", "Name", "Script")
	biz.ErrIsNil(err, "script not found")

	script := ms.Script
	// 如果有脚本参数，则用脚本参数替换脚本中的模板占位符参数
	if params := rc.Query("params"); params != "" {
		script, err = stringx.TemplateParse(ms.Script, jsonx.ToMap(params))
		biz.ErrIsNilAppendErr(err, "failed to parse the script template parameter: %s")
	}
	cli, err := m.MachineApp.GetCliByAc(ac)
	biz.ErrIsNilAppendErr(err, "connection error: %s")
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.CodePath...), "%s")

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
