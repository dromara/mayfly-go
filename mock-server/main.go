package main

import (
	"flag"
	"fmt"
	"mayfly-go/base/rediscli"
	"mayfly-go/base/utils/yml"
	_ "mayfly-go/mock-server/routers"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/go-redis/redis"
	// _ "github.com/go-sql-driver/mysql"
)

// 启动配置参数
type StartConfigParam struct {
	ConfigFilePath string // 配置文件路径
}

// yaml配置文件映射对象
type Config struct {
	Server struct {
		Port int `yaml:"port"`
	}
	Redis struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		Db       int    `yaml:"db"`
	}
}

// 启动可执行文件时的参数
var startConfigParam *StartConfigParam

// 配置文件映射对象
var ymlConfig Config

// 获取执行可执行文件时，指定的启动参数
func getStartConfig() *StartConfigParam {
	configFilePath := flag.String("e", "./config.yml", "配置文件路径，默认为可执行文件目录")
	flag.Parse()
	// 获取配置文件绝对路径
	path, _ := filepath.Abs(*configFilePath)
	sc := &StartConfigParam{ConfigFilePath: path}
	return sc
}

func init() {
	configFilePath := flag.String("e", "./config.yml", "配置文件路径，默认为可执行文件目录")
	flag.Parse()
	// 获取启动参数中，配置文件的绝对路径
	path, _ := filepath.Abs(*configFilePath)
	startConfigParam = &StartConfigParam{ConfigFilePath: path}
	// 读取配置文件信息
	yc := &Config{}
	if err := yml.LoadYml(startConfigParam.ConfigFilePath, yc); err != nil {
		panic(fmt.Sprintf("读取配置文件[%s]失败: %s", startConfigParam.ConfigFilePath, err.Error()))
	}
	ymlConfig = *yc
}

func main() {
	// 设置redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", ymlConfig.Redis.Host, ymlConfig.Redis.Port),
		Password: ymlConfig.Redis.Password, // no password set
		DB:       ymlConfig.Redis.Db,       // use default DB
	})
	rediscli.SetCli(rdb)

	web.InsertFilter("/*", web.BeforeRouter, TransparentStatic)
	// 跨域配置
	web.InsertFilter("/**", web.BeforeRouter, cors.Allow(&cors.Options{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		AllowCredentials: true,
	}))
	web.Run()
}

// 解决beego无法访问根目录静态文件
func TransparentStatic(ctx *context.Context) {
	if strings.Index(ctx.Request.URL.Path, "api/") >= 0 {
		return
	}
	http.ServeFile(ctx.ResponseWriter, ctx.Request, "static/"+ctx.Request.URL.Path)
}
