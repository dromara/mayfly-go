package starter

import (
	"mayfly-go/initialize"
	"mayfly-go/migrations"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/global"
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

	// 数据库升级操作
	if err := migrations.RunMigrations(global.Db); err != nil {
		global.Log.Fatalf("数据库升级失败: %v", err)
	}

	// 初始化其他需要启动时运行的方法
	initialize.InitOther()

	// 运行web服务
	runWebServer()
}
