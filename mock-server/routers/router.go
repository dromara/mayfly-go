package routers

import (
	"mayfly-go/mock-server/controllers"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Include(&controllers.MockController{})

	mock := &controllers.MockController{}
	web.Router("/api/mock-datas", mock, "post:CreateMockData")
	web.Router("/api/mock-datas/?:method", mock, "get:GetMockData")
	web.Router("/api/mock-datas", mock, "put:UpdateMockData")
	web.Router("/api/mock-datas/?:method", mock, "delete:DeleteMockData")
}
