package starter

import (
	"context"
	"mayfly-go/initialize"
	"mayfly-go/internal/pkg/config"
	"mayfly-go/migration"
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

	// 初始化缓存
	initCache()

	// 数据库升级操作
	if err := migration.RunMigrations(global.Db); err != nil {
		logx.Panicf("db migration failed: %v", err)
	}

	// 参数校验器初始化、如错误提示中文转译等
	validatorx.Init()
	// 注册自定义正则表达式校验规则
	RegisterCustomPatterns()

	// 初始化其他需要启动时运行的方法
	initialize.InitOther()

	// 运行web服务
	runWebServer(ctx)
}

// 注册自定义正则表达式校验规则
func RegisterCustomPatterns() {
	// 账号用户名校验
	validatorx.RegisterPattern("account_username", "^[a-zA-Z0-9_]{5,20}$", "只允许输入5-20位大小写字母、数字、下划线")
	validatorx.RegisterPattern("resource_code", "^[a-zA-Z0-9_\\-.:]{1,32}$", "只允许输入1-32位大小写字母、数字、_-.:")
}
