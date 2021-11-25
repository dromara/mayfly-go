package api

import (
	"bytes"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/utils"
	"mayfly-go/base/ws"
	"mayfly-go/server/devops/api/form"
	"mayfly-go/server/devops/api/vo"
	"mayfly-go/server/devops/application"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/infrastructure/machine"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Machine struct {
	MachineApp application.Machine
}

func (m *Machine) Machines(rc *ctx.ReqCtx) {
	res := m.MachineApp.GetMachineList(new(entity.Machine), ginx.GetPageParam(rc.GinCtx), new([]*vo.MachineVO))
	list := res.List.(*[]*vo.MachineVO)
	for _, mv := range *list {
		mv.HasCli = machine.HasCli(*mv.Id)
	}
	rc.ResData = res
}

func (m *Machine) MachineStats(rc *ctx.ReqCtx) {
	writer := bytes.NewBufferString("")
	stats := m.MachineApp.GetCli(GetMachineId(rc.GinCtx)).GetAllStats()
	machine.ShowStats(writer, stats)
	rc.ResData = writer.String()
}

func (m *Machine) SaveMachine(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	machineForm := new(form.MachineForm)
	ginx.BindJsonAndValid(g, machineForm)

	entity := new(entity.Machine)
	utils.Copy(entity, machineForm)

	entity.SetBaseInfo(rc.LoginAccount)
	m.MachineApp.Save(entity)
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
		panic(biz.NewBizErr("没有权限"))
	}
	// 演示环境禁止非admin用户执行
	// if rc.LoginAccount.Username != "admin" {
	// 	panic(biz.NewBizErrCode(401, "非admin用户无权该操作"))
	// }

	cols := ginx.QueryInt(g, "cols", 80)
	rows := ginx.QueryInt(g, "rows", 40)

	sws, err := machine.NewLogicSshWsSession(cols, rows, m.MachineApp.GetCli(GetMachineId(g)), wsConn)
	biz.ErrIsNilAppendErr(err, "连接失败：%s")
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
