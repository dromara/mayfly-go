package main

import (
	"mayfly-go/base/global"
	"mayfly-go/base/starter"
	"mayfly-go/devops/initialize"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	global.Db = starter.GormMysql()
	starter.RunWebServer(initialize.InitRouter())
}
