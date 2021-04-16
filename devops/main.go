package main

import (
	"mayfly-go/base/global"
	"mayfly-go/base/initialize"
	_ "mayfly-go/devops/routers"
	"mayfly-go/mock-server/starter"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db := initialize.GormMysql()
	if db == nil {
		global.Log.Panic("mysql连接失败")
	} else {
		global.Db = db
	}

	starter.RunServer()
}
