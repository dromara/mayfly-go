package main

import (
	"mayfly-go/base/ctx"
	"mayfly-go/base/global"
	"mayfly-go/base/starter"
	"mayfly-go/server/initialize"
)

func main() {
	ctx.UseBeforeHandlerInterceptor(ctx.PermissionHandler)
	ctx.UseAfterHandlerInterceptor(ctx.LogHandler)
	global.Db = starter.GormMysql()
	starter.RunWebServer(initialize.InitRouter())
}
