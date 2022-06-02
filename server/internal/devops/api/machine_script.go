package api

import (
	"fmt"
	"mayfly-go/internal/devops/api/form"
	"mayfly-go/internal/devops/api/vo"
	"mayfly-go/internal/devops/application"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MachineScript struct {
	MachineScriptApp application.MachineScript
	MachineApp       application.Machine
	ProjectApp       application.Project
}

func (m *MachineScript) MachineScripts(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	condition := &entity.MachineScript{MachineId: GetMachineId(g)}
	rc.ResData = m.MachineScriptApp.GetPageList(condition, ginx.GetPageParam(g), new([]vo.MachineScriptVO))
}

func (m *MachineScript) SaveMachineScript(rc *ctx.ReqCtx) {
	form := new(form.MachineScriptForm)
	ginx.BindJsonAndValid(rc.GinCtx, form)
	rc.ReqParam = form

	// 转换为entity，并设置基本信息
	machineScript := new(entity.MachineScript)
	utils.Copy(machineScript, form)
	machineScript.SetBaseInfo(rc.LoginAccount)

	m.MachineScriptApp.Save(machineScript)
}

func (m *MachineScript) DeleteMachineScript(rc *ctx.ReqCtx) {
	msa := m.MachineScriptApp
	sid := GetMachineScriptId(rc.GinCtx)
	ms := msa.GetById(sid)
	biz.NotNil(ms, "该脚本不存在")
	rc.ReqParam = fmt.Sprintf("[scriptId: %d, name: %s, desc: %s, script: %s]", sid, ms.Name, ms.Description, ms.Script)
	msa.Delete(sid)
}

func (m *MachineScript) RunMachineScript(rc *ctx.ReqCtx) {
	g := rc.GinCtx

	scriptId := GetMachineScriptId(g)
	machineId := GetMachineId(g)
	ms := m.MachineScriptApp.GetById(scriptId, "MachineId", "Name", "Script")
	biz.NotNil(ms, "该脚本不存在")
	biz.IsTrue(ms.MachineId == application.Common_Script_Machine_Id || ms.MachineId == machineId, "该脚本不属于该机器")

	script := ms.Script
	// 如果有脚本参数，则用脚本参数替换脚本中的模板占位符参数
	if params := g.Query("params"); params != "" {
		script = utils.TemplateParse(ms.Script, utils.Json2Map(params))
	}
	cli := m.MachineApp.GetCli(machineId)
	biz.ErrIsNilAppendErr(m.ProjectApp.CanAccess(rc.LoginAccount.Id, cli.GetMachine().ProjectId), "%s")

	res, err := cli.Run(script)
	// 记录请求参数
	rc.ReqParam = fmt.Sprintf("[machineId: %d, scriptId: %d, name: %s]", machineId, scriptId, ms.Name)
	if err != nil {
		panic(biz.NewBizErr(fmt.Sprintf("执行命令失败：%s", err.Error())))
	}
	rc.ResData = res
}

func GetMachineScriptId(g *gin.Context) uint64 {
	scriptId, _ := strconv.Atoi(g.Param("scriptId"))
	biz.IsTrue(scriptId > 0, "scriptId错误")
	return uint64(scriptId)
}
