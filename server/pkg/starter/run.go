package starter

import (
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/logger"
	"mayfly-go/pkg/req"
)

func RunWebServer() {
	// 初始化config.yml配置文件映射信息
	config.Init()
	// 初始化日志配置信息
	logger.Init()
	// 初始化jwt key与expire time等
	req.InitTokenConfig()

	// 打印banner
	printBanner()
	// 初始化并赋值数据库全局变量
	initDb()
	// 有配置redis信息，则初始化redis。多台机器部署需要使用redis存储验证码、权限、公私钥等
	initRedis()
	// 运行web服务
	runWebServer()
}
