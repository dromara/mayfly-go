package main

import (
	"fmt"
	"mayfly-go/base/global"
	"mayfly-go/base/rediscli"
	"mayfly-go/mock-server/initialize"
	_ "mayfly-go/mock-server/routers"
	"mayfly-go/mock-server/starter"

	"github.com/go-redis/redis"
	// _ "github.com/go-sql-driver/mysql"
)

func main() {
	// 设置redis客户端
	redisConf := global.Config.Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisConf.Host, redisConf.Port),
		Password: redisConf.Password, // no password set
		DB:       redisConf.Db,       // use default DB
	})
	// 测试连接
	_, e := rdb.Ping().Result()
	if e != nil {
		global.Log.Panic(fmt.Sprintf("连接redis失败! [%s:%d]", redisConf.Host, redisConf.Port))
	}
	rediscli.SetCli(rdb)

	db := initialize.GormMysql()
	if db == nil {
		global.Log.Panic("mysql连接失败")
	} else {
		global.Db = db
	}

	starter.RunServer()
}
