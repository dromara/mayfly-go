package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context/param"
)

func init() {

    beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"] = append(beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"],
        beego.ControllerComments{
            Method: "UpdateMockData",
            Router: "/api/mock-datas",
            AllowHTTPMethods: []string{"put"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"] = append(beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"],
        beego.ControllerComments{
            Method: "CreateMockData",
            Router: "/api/mock-datas",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"] = append(beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"],
        beego.ControllerComments{
            Method: "GetAllData",
            Router: "/api/mock-datas",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"] = append(beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"],
        beego.ControllerComments{
            Method: "GetMockData",
            Router: "/api/mock-datas/:method",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"] = append(beego.GlobalControllerRouter["mayfly-go/mock-server/controllers:MockController"],
        beego.ControllerComments{
            Method: "DeleteMockData",
            Router: "/api/mock-datas/:method",
            AllowHTTPMethods: []string{"delete"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

}
