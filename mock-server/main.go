package main

import (
	"mayfly-go/base/rediscli"
	"mayfly-go/base/starter"
	"mayfly-go/mock-server/initialize"
	// _ "github.com/go-sql-driver/mysql"
)

func main() {
	rediscli.SetCli(starter.ConnRedis())
	starter.RunWebServer(initialize.InitRouter())
}
