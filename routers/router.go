package routers

import (
	"github.com/astaxie/beego"
	"mayfly-go/controllers"
)

func init() {
	//beego.Router("/account/login", &controllers.LoginController{})
	//beego.Router("/account", &controllers.AccountController{})
	//beego.Include(&controllers.AccountController{})
	//beego.Include()
	beego.Router("/api/accounts/login", &controllers.AccountController{}, "post:Login")
	beego.Router("/api/accounts", &controllers.AccountController{}, "get:Accounts")

	machine := &controllers.MachineController{}
	beego.Router("/api/machines", machine, "get:Machines")
	beego.Router("/api/machines/?:machineId/run", machine, "get:Run")
	beego.Router("/api/machines/?:machineId/top", machine, "get:Top")
	beego.Router("/api/machines/?:machineId/sysinfo", machine, "get:SysInfo")
	beego.Router("/api/machines/?:machineId/process", machine, "get:GetProcessByName")
	//beego.Router("/machines/?:machineId/ws", machine, "get:WsSSH")
}
