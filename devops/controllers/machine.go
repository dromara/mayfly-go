package controllers

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/devops/machine"
	"mayfly-go/devops/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Machines(rc *ctx.ReqCtx) {
	rc.ResData = models.GetMachineList(ginx.GetPageParam(rc.GinCtx))
}

func Run(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	cmd := g.Query("cmd")
	biz.NotEmpty(cmd, "cmd不能为空")

	rc.ReqParam = cmd

	res, err := getCli(g).Run(cmd)
	biz.BizErrIsNil(err, "执行命令失败")
	rc.ResData = res
}

// 系统基本信息
func SysInfo(rc *ctx.ReqCtx) {
	res, err := getCli(rc.GinCtx).GetSystemInfo()
	biz.BizErrIsNil(err, "获取系统基本信息失败")
	rc.ResData = res
}

// top命令信息
func Top(rc *ctx.ReqCtx) {
	rc.ResData = getCli(rc.GinCtx).GetTop()
}

func GetProcessByName(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	name := g.Query("name")
	biz.NotEmpty(name, "name不能为空")
	res, err := getCli(g).GetProcessByName(name)
	biz.BizErrIsNil(err, "获取失败")
	rc.ResData = res
}

func WsSSH(g *gin.Context) {
	wsConn, err := upgrader.Upgrade(g.Writer, g.Request, nil)
	if err != nil {
		panic(biz.NewBizErr("获取requst responsewirte错误"))
	}

	cols := ginx.QueryInt(g, "cols", 80)
	rows := ginx.QueryInt(g, "rows", 40)

	sws, err := machine.NewLogicSshWsSession(cols, rows, getCli(g), wsConn)
	if sws == nil {
		panic(biz.NewBizErr("连接失败"))
	}
	//if wshandleError(wsConn, err) {
	//	return
	//}
	defer sws.Close()

	quitChan := make(chan bool, 3)
	sws.Start(quitChan)
	go sws.Wait(quitChan)

	<-quitChan
}

func GetMachineId(g *gin.Context) uint64 {
	machineId, _ := strconv.Atoi(g.Param("machineId"))
	biz.IsTrue(machineId > 0, "machineId错误")
	return uint64(machineId)
}

func getCli(g *gin.Context) *machine.Cli {
	cli, err := machine.GetCli(GetMachineId(g))
	biz.BizErrIsNil(err, "获取客户端错误")
	return cli
}
