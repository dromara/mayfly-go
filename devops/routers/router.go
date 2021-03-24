package routers

import (
	"mayfly-go/devops/controllers"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	//web.Router("/account/login", &controllers.LoginController{})
	//web.Router("/account", &controllers.AccountController{})
	//web.Include(&controllers.AccountController{})
	//web.Include()
	web.Router("/api/accounts/login", &controllers.AccountController{}, "post:Login")
	web.Router("/api/accounts", &controllers.AccountController{}, "get:Accounts")

	machine := &controllers.MachineController{}
	web.Router("/api/machines", machine, "get:Machines")
	web.Router("/api/machines/?:machineId/run", machine, "get:Run")
	web.Router("/api/machines/?:machineId/top", machine, "get:Top")
	web.Router("/api/machines/?:machineId/sysinfo", machine, "get:SysInfo")
	web.Router("/api/machines/?:machineId/process", machine, "get:GetProcessByName")
	web.Router("/api/machines/?:machineId/terminal", machine, "get:WsSSH")

	web.Include(&controllers.DbController{})
	// db := &controllers.DbController{}
	// web.Router("/api/dbs", db, "get:Dbs")
	// web.Router("/api/db/?:dbId/select", db, "get:SelectData")
	// web.Router("/api/db/?:dbId/t-metadata", db, "get:TableMA")
	// web.Router("/api/db/?:dbId/c-metadata", db, "get:ColumnMA")
	// web.Router("/api/db/?:dbId/hint-tables", db, "get:HintTables")
	// web.Router("/api/db/?:dbId/sql", db, "post:SaveSql")
	// web.Router("/api/db/?:dbId/sql", db, "get:GetSql")
}
