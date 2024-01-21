package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type MachineScript struct {
	MachineScriptApp application.MachineScript `inject:""`
	MachineApp       application.Machine       `inject:""`
	TagApp           tagapp.TagTree            `inject:"TagTreeApp"`
}

func (m *MachineScript) MachineScripts(rc *req.Ctx) {
	g := rc.GinCtx
	condition := &entity.MachineScript{MachineId: GetMachineId(g)}
	res, err := m.MachineScriptApp.GetPageList(condition, ginx.GetPageParam(g), new([]vo.MachineScriptVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *MachineScript) SaveMachineScript(rc *req.Ctx) {
	form := new(form.MachineScriptForm)
	machineScript := ginx.BindJsonAndCopyTo(rc.GinCtx, form, new(entity.MachineScript))

	rc.ReqParam = form
	biz.ErrIsNil(m.MachineScriptApp.Save(rc.MetaCtx, machineScript))
}

func (m *MachineScript) DeleteMachineScript(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "scriptId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		m.MachineScriptApp.Delete(rc.MetaCtx, uint64(value))
	}
}

func (m *MachineScript) RunMachineScript(rc *req.Ctx) {
	g := rc.GinCtx

	scriptId := GetMachineScriptId(g)
	machineId := GetMachineId(g)
	ms, err := m.MachineScriptApp.GetById(new(entity.MachineScript), scriptId, "MachineId", "Name", "Script")
	biz.ErrIsNil(err, "该脚本不存在")
	biz.IsTrue(ms.MachineId == application.Common_Script_Machine_Id || ms.MachineId == machineId, "该脚本不属于该机器")

	script := ms.Script
	// 如果有脚本参数，则用脚本参数替换脚本中的模板占位符参数
	if params := g.Query("params"); params != "" {
		script, err = stringx.TemplateParse(ms.Script, jsonx.ToMap(params))
		biz.ErrIsNilAppendErr(err, "脚本模板参数解析失败: %s")
	}
	cli, err := m.MachineApp.GetCli(machineId)
	biz.ErrIsNilAppendErr(err, "获取客户端连接失败: %s")
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.GetLoginAccount().Id, cli.Info.TagPath...), "%s")

	res, err := cli.Run(script)
	// 记录请求参数
	rc.ReqParam = collx.Kvs("machine", cli.Info, "scriptId", scriptId, "name", ms.Name)
	if res == "" {
		biz.ErrIsNilAppendErr(err, "执行命令失败：%s")
	}
	rc.ResData = res
}

func GetMachineScriptId(g *gin.Context) uint64 {
	scriptId, _ := strconv.Atoi(g.Param("scriptId"))
	biz.IsTrue(scriptId > 0, "scriptId错误")
	return uint64(scriptId)
}
