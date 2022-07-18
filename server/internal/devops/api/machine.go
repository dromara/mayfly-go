package api

import (
	"fmt"
	"mayfly-go/internal/devops/api/form"
	"mayfly-go/internal/devops/api/vo"
	"mayfly-go/internal/devops/application"
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/infrastructure/machine"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/utils"
	"mayfly-go/pkg/ws"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Machine struct {
	MachineApp application.Machine
	ProjectApp application.Project
}

func (m *Machine) Machines(rc *ctx.ReqCtx) {
	condition := new(entity.Machine)
	// 使用创建者id模拟账号成员id
	condition.CreatorId = rc.LoginAccount.Id
	condition.Ip = rc.GinCtx.Query("ip")
	condition.Name = rc.GinCtx.Query("name")
	condition.ProjectId = uint64(ginx.QueryInt(rc.GinCtx, "projectId", 0))

	res := m.MachineApp.GetMachineList(condition, ginx.GetPageParam(rc.GinCtx), new([]*vo.MachineVO))
	if res.Total == 0 {
		rc.ResData = res
		return
	}

	list := res.List.(*[]*vo.MachineVO)
	for _, mv := range *list {
		mv.HasCli = machine.HasCli(*mv.Id)
	}
	rc.ResData = res
}

func (m *Machine) MachineStats(rc *ctx.ReqCtx) {
	stats := m.MachineApp.GetCli(GetMachineId(rc.GinCtx)).GetAllStats()
	rc.ResData = stats
}

func (m *Machine) SaveMachine(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	machineForm := new(form.MachineForm)
	ginx.BindJsonAndValid(g, machineForm)

	entity := new(entity.Machine)
	utils.Copy(entity, machineForm)

	// 密码解密，并使用解密后的赋值
	originPwd, err := utils.DefaultRsaDecrypt(machineForm.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	entity.Password = originPwd

	// 密码脱敏记录日志
	machineForm.Password = "****"
	rc.ReqParam = machineForm

	entity.SetBaseInfo(rc.LoginAccount)
	m.MachineApp.Save(entity)
}

func (m *Machine) ChangeStatus(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	id := uint64(ginx.PathParamInt(g, "machineId"))
	status := int8(ginx.PathParamInt(g, "status"))
	rc.ReqParam = fmt.Sprintf("id: %d -- status: %d", id, status)
	m.MachineApp.ChangeStatus(id, status)
}

func (m *Machine) DeleteMachine(rc *ctx.ReqCtx) {
	id := uint64(ginx.PathParamInt(rc.GinCtx, "machineId"))
	rc.ReqParam = id
	m.MachineApp.Delete(id)
}

// 关闭机器客户端
func (m *Machine) CloseCli(rc *ctx.ReqCtx) {
	machine.DeleteCli(GetMachineId(rc.GinCtx))
}

// 获取进程列表信息
func (m *Machine) GetProcess(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	cmd := "ps -aux "
	sortType := g.Query("sortType")
	if sortType == "2" {
		cmd += "--sort -pmem "
	} else {
		cmd += "--sort -pcpu "
	}

	pname := g.Query("name")
	if pname != "" {
		cmd += fmt.Sprintf("| grep %s ", pname)
	}

	count := g.Query("count")
	if count == "" {
		count = "10"
	}

	cmd += "| head -n " + count

	cli := m.MachineApp.GetCli(GetMachineId(rc.GinCtx))
	biz.ErrIsNilAppendErr(m.ProjectApp.CanAccess(rc.LoginAccount.Id, cli.GetMachine().ProjectId), "%s")

	res, err := cli.Run(cmd)
	biz.ErrIsNilAppendErr(err, "获取进程信息失败: %s")
	rc.ResData = res
}

// 终止进程
func (m *Machine) KillProcess(rc *ctx.ReqCtx) {
	pid := rc.GinCtx.Query("pid")
	biz.NotEmpty(pid, "进程id不能为空")

	cli := m.MachineApp.GetCli(GetMachineId(rc.GinCtx))
	biz.ErrIsNilAppendErr(m.ProjectApp.CanAccess(rc.LoginAccount.Id, cli.GetMachine().ProjectId), "%s")

	_, err := cli.Run("kill -9 " + pid)
	biz.ErrIsNilAppendErr(err, "终止进程失败: %s")
}

func (m *Machine) WsSSH(g *gin.Context) {
	wsConn, err := ws.Upgrader.Upgrade(g.Writer, g.Request, nil)
	defer func() {
		if err := recover(); err != nil {
			wsConn.WriteMessage(websocket.TextMessage, []byte(err.(error).Error()))
			wsConn.Close()
		}
	}()

	if err != nil {
		panic(biz.NewBizErr("升级websocket失败"))
	}
	// 权限校验
	rc := ctx.NewReqCtxWithGin(g).WithRequiredPermission(ctx.NewPermission("machine:terminal"))
	if err = ctx.PermissionHandler(rc); err != nil {
		panic(biz.NewBizErr("\033[1;31m您没有权限操作该机器终端,请重新登录后再试~\033[0m"))
	}

	cols := ginx.QueryInt(g, "cols", 80)
	rows := ginx.QueryInt(g, "rows", 40)

	cli := m.MachineApp.GetCli(GetMachineId(g))
	biz.ErrIsNilAppendErr(m.ProjectApp.CanAccess(rc.LoginAccount.Id, cli.GetMachine().ProjectId), "%s")

	sws, err := machine.NewLogicSshWsSession(cols, rows, cli, wsConn)
	biz.ErrIsNilAppendErr(err, "\033[1;31m连接失败：%s\033[0m")
	defer sws.Close()

	quitChan := make(chan bool, 3)
	sws.Start(quitChan)
	go sws.Wait(quitChan)

	<-quitChan
}

func GetMachineId(g *gin.Context) uint64 {
	machineId, _ := strconv.Atoi(g.Param("machineId"))
	biz.IsTrue(machineId != 0, "machineId错误")
	return uint64(machineId)
}
