package api

import (
	"fmt"
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type MachineScript struct {
	MachineScriptApp application.MachineScript
	MachineApp       application.Machine
	TagApp           tagapp.TagTree
}

func (m *MachineScript) MachineScripts(rc *req.Ctx) {
	g := rc.GinCtx
	condition := &entity.MachineScript{MachineId: GetMachineId(g)}
	rc.ResData = m.MachineScriptApp.GetPageList(condition, ginx.GetPageParam(g), new([]vo.MachineScriptVO))
}

func (m *MachineScript) SaveMachineScript(rc *req.Ctx) {
	form := new(form.MachineScriptForm)
	ginx.BindJsonAndValid(rc.GinCtx, form)
	rc.ReqParam = form

	// 转换为entity，并设置基本信息
	machineScript := new(entity.MachineScript)
	utils.Copy(machineScript, form)
	machineScript.SetBaseInfo(rc.LoginAccount)

	m.MachineScriptApp.Save(machineScript)
}

func (m *MachineScript) DeleteMachineScript(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "scriptId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		m.MachineScriptApp.Delete(uint64(value))
	}
}

func (m *MachineScript) RunMachineScript(rc *req.Ctx) {
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
	biz.ErrIsNilAppendErr(m.TagApp.CanAccess(rc.LoginAccount.Id, cli.GetMachine().TagPath), "%s")

	res, err := cli.Run(script)
	// 记录请求参数
	rc.ReqParam = fmt.Sprintf("[machine: %s, scriptId: %d, name: %s]", cli.GetMachine().GetLogDesc(), scriptId, ms.Name)
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
