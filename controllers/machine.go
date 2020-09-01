package controllers

import (
	"github.com/gorilla/websocket"
	"mayfly-go/base"
	"mayfly-go/machine"
	"mayfly-go/models"
	"net/http"
	"strconv"
)

type MachineController struct {
	base.Controller
}

var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024 * 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *MachineController) Machines() {
	c.ReturnData(true, func(account *base.LoginAccount) interface{} {
		return models.GetMachineList(c.GetPageParam())
	})
}

func (c *MachineController) Run() {
	c.ReturnData(true, func(account *base.LoginAccount) interface{} {
		cmd := c.GetString("cmd")
		base.NotEmpty(cmd, "cmd不能为空")

		return machine.GetCli(c.GetMachineId()).Run(cmd)
	})
}

// 系统基本信息
func (c *MachineController) SysInfo() {
	c.ReturnData(true, func(account *base.LoginAccount) interface{} {
		return machine.GetSystemInfo(machine.GetCli(c.GetMachineId()))
	})
}

// top命令信息
func (c *MachineController) Top() {
	c.ReturnData(true, func(account *base.LoginAccount) interface{} {
		return machine.GetTop(machine.GetCli(c.GetMachineId()))
	})
}

func (c *MachineController) GetProcessByName() {
	c.ReturnData(true, func(account *base.LoginAccount) interface{} {
		name := c.GetString("name")
		base.NotEmpty(name, "name不能为空")
		return machine.GetProcessByName(machine.GetCli(c.GetMachineId()), name)
	})
}

//func (c *MachineController) WsSSH() {
//	wsConn, err := upGrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
//	if err != nil {
//		panic(base.NewBizErr("获取requst responsewirte错误"))
//	}
//
//	cols, _ := c.GetInt("col", 80)
//	rows, _ := c.GetInt("rows", 40)
//
//	sws, err := machine.NewLogicSshWsSession(cols, rows, true, machine.GetCli(c.GetMachineId()), wsConn)
//	if sws == nil {
//		panic(base.NewBizErr("连接失败"))
//	}
//	//if wshandleError(wsConn, err) {
//	//	return
//	//}
//	defer sws.Close()
//
//	quitChan := make(chan bool, 3)
//	sws.Start(quitChan)
//	go sws.Wait(quitChan)
//
//	<-quitChan
//}

func (c *MachineController) GetMachineId() uint64 {
	machineId, _ := strconv.Atoi(c.Ctx.Input.Param(":machineId"))
	base.IsTrue(machineId > 0, "machineId错误")
	return uint64(machineId)
}
