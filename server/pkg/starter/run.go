package starter

import (
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/logger"
)

func RunWebServer() {
	// 初始化config.yml配置文件映射信息
	config.Init()
	// 初始化日志配置信息
	logger.Init()
	// 初始化jwt key与expire time等
	ctx.InitTokenConfig()

	// 打印banner
	printBanner()
	// 初始化并赋值数据库全局变量
	initDb()
	// 运行web服务
	runWebServer()
}
