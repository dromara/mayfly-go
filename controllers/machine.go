package controllers

import (
	"mayfly-go/base"
	"mayfly-go/base/ctx"
	"mayfly-go/base/model"
	"mayfly-go/machine"
	"mayfly-go/models"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

type MachineController struct {
	base.Controller
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *MachineController) Machines() {
	c.ReturnData(true, func(account *ctx.LoginAccount) interface{} {
		return models.GetMachineList(c.GetPageParam())
	})
}

func (c *MachineController) Run() {
	c.ReturnData(true, func(account *ctx.LoginAccount) interface{} {
		cmd := c.GetString("cmd")
		model.NotEmpty(cmd, "cmd不能为空")

		res, err := c.getCli().Run(cmd)
		model.BizErrIsNil(err, "执行命令失败")
		return res
	})
}

// 系统基本信息
func (c *MachineController) SysInfo() {
	c.ReturnData(true, func(account *ctx.LoginAccount) interface{} {
		res, err := c.getCli().GetSystemInfo()
		model.BizErrIsNil(err, "获取系统基本信息失败")
		return res
	})
}

// top命令信息
func (c *MachineController) Top() {
	c.ReturnData(true, func(account *ctx.LoginAccount) interface{} {
		return c.getCli().GetTop()
	})
}

func (c *MachineController) GetProcessByName() {
	c.ReturnData(true, func(account *ctx.LoginAccount) interface{} {
		name := c.GetString("name")
		model.NotEmpty(name, "name不能为空")
		res, err := c.getCli().GetProcessByName(name)
		model.BizErrIsNil(err, "获取失败")
		return res
	})
}

func (c *MachineController) WsSSH() {
	wsConn, err := upgrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
	if err != nil {
		panic(model.NewBizErr("获取requst responsewirte错误"))
	}

	cols, _ := c.GetInt("cols", 80)
	rows, _ := c.GetInt("rows", 40)

	sws, err := machine.NewLogicSshWsSession(cols, rows, c.getCli(), wsConn)
	if sws == nil {
		panic(model.NewBizErr("连接失败"))
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

func (c *MachineController) GetMachineId() uint64 {
	machineId, _ := strconv.Atoi(c.Ctx.Input.Param(":machineId"))
	model.IsTrue(machineId > 0, "machineId错误")
	return uint64(machineId)
}

func (c *MachineController) getCli() *machine.Cli {
	cli, err := machine.GetCli(c.GetMachineId())
	model.BizErrIsNil(err, "获取客户端错误")
	return cli
}
