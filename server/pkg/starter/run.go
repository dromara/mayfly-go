package starter

import (
	"context"
	"mayfly-go/initialize"
	"mayfly-go/migrations"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/validatorx"
	"os"
	"os/signal"
	"syscall"
)

func RunWebServer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	// 初始化config.yml配置文件映射信息或使用环境变量。并初始化系统日志相关配置
	config.Init()

	// 打印banner
	printBanner()

	// 初始化并赋值数据库全局变量
	initDb()

	// 有配置redis信息，则初始化redis。多台机器部署需要使用redis存储验证码、权限、公私钥等
	initRedis()

	// 数据库升级操作
	if err := migrations.RunMigrations(global.Db); err != nil {
		logx.Panicf("数据库升级失败: %v", err)
	}

	// 参数校验器初始化、如错误提示中文转译等
	validatorx.Init()

	// 初始化其他需要启动时运行的方法
	initialize.InitOther()

	// 运行web服务
	runWebServer(ctx)
}
