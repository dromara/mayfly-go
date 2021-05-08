package apis

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/utils"
	"mayfly-go/devops/apis/form"
	"mayfly-go/devops/apis/vo"
	"mayfly-go/devops/application"
	"mayfly-go/devops/domain/entity"
	"mayfly-go/devops/infrastructure/machine"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var WsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Machine struct {
	MachineApp application.IMachine
}

func (m *Machine) Machines(rc *ctx.ReqCtx) {
	rc.ResData = m.MachineApp.GetMachineList(new(entity.Machine), ginx.GetPageParam(rc.GinCtx), new([]vo.MachineVO))
}

func (m *Machine) SaveMachine(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	machineForm := new(form.MachineForm)
	ginx.BindJsonAndValid(g, machineForm)

	entity := new(entity.Machine)
	utils.Copy(entity, machineForm)
	biz.ErrIsNil(machine.TestConn(entity), "无法连接")

	entity.SetBaseInfo(rc.LoginAccount)
	m.MachineApp.Save(entity)
}

// top命令信息
func (m *Machine) Top(rc *ctx.ReqCtx) {
	rc.ResData = m.MachineApp.GetCli(GetMachineId(rc.GinCtx)).GetTop()
}

func (m *Machine) WsSSH(g *gin.Context) {
	wsConn, err := WsUpgrader.Upgrade(g.Writer, g.Request, nil)
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
	if err = ctx.PermissionHandler(ctx.NewReqCtxWithGin(g)); err != nil {
		panic(biz.NewBizErr("没有权限"))
	}

	cols := ginx.QueryInt(g, "cols", 80)
	rows := ginx.QueryInt(g, "rows", 40)

	sws, err := machine.NewLogicSshWsSession(cols, rows, m.MachineApp.GetCli(GetMachineId(g)), wsConn)
	if sws == nil {
		panic(biz.NewBizErr("连接失败"))
	}
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
