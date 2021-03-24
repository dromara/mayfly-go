package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context/param"
)

func init() {

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:AccountController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:AccountController"],
        beego.ControllerComments{
            Method: "Accounts",
            Router: "/accounts",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:AccountController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:AccountController"],
        beego.ControllerComments{
            Method: "Login",
            Router: "/accounts/login",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"],
        beego.ControllerComments{
            Method: "ColumnMA",
            Router: "/api/db/:dbId/c-metadata",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"],
        beego.ControllerComments{
            Method: "ExecSql",
            Router: "/api/db/:dbId/exec-sql",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"],
        beego.ControllerComments{
            Method: "HintTables",
            Router: "/api/db/:dbId/hint-tables",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"],
        beego.ControllerComments{
            Method: "SelectData",
            Router: "/api/db/:dbId/select",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"],
        beego.ControllerComments{
            Method: "SaveSql",
            Router: "/api/db/:dbId/sql",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"],
        beego.ControllerComments{
            Method: "GetSql",
            Router: "/api/db/:dbId/sql",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"],
        beego.ControllerComments{
            Method: "TableMA",
            Router: "/api/db/:dbId/t-metadata",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"] = append(beego.GlobalControllerRouter["mayfly-go/devops/controllers:DbController"],
        beego.ControllerComments{
            Method: "Dbs",
            Router: "/api/dbs",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

}
