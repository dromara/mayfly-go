package main

import (
	"fmt"
	_ "mayfly-go/internal/ai/init"
	_ "mayfly-go/internal/auth/init"
	_ "mayfly-go/internal/common/init"
	_ "mayfly-go/internal/db/init"
	_ "mayfly-go/internal/docker/init"
	_ "mayfly-go/internal/es/init"
	_ "mayfly-go/internal/file/init"
	_ "mayfly-go/internal/flow/init"
	_ "mayfly-go/internal/machine/init"
	_ "mayfly-go/internal/mongo/init"
	_ "mayfly-go/internal/msg/init"
	"mayfly-go/internal/pkg/config"
	_ "mayfly-go/internal/redis/init"
	_ "mayfly-go/internal/sys/init"
	_ "mayfly-go/internal/tag/init"
	"mayfly-go/migration"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/starter"
	"mayfly-go/static"
	"os"
	"runtime/debug"

	sysapp "mayfly-go/internal/sys/application"

	"gorm.io/gorm"
)

func main() {
	// 初始化config.yml配置文件映射信息或使用环境变量
	config, err := config.Init()
	if err != nil {
		logx.Panicf("config init failed: %v", err)
	}
	printBanner()

	if err = starter.Run(config.Conf,
		starter.WithOnDbReady(func(db *gorm.DB) error {
			// 数据库升级操作
			return migration.RunMigrations(db)
		}),
		starter.WithLogSaver(func() req.SaveLogFunc {
			// 日志保存
			return sysapp.GetSyslogApp().SaveFromReq
		}),
		starter.WithStaticRouter(static.Router()),
	); err != nil {
		logx.Panicf("starter server failed: %v", err)
	}
}

func printBanner() {
	buildInfo, _ := debug.ReadBuildInfo()
	logx.Print(fmt.Sprintf(`
                        __ _                         
 _ __ ___   __ _ _   _ / _| |_   _        __ _  ___  
| '_ ' _ \ / _' | | | | |_| | | | |_____ / _' |/ _ \ 
| | | | | | (_| | |_| |  _| | |_| |_____| (_| | (_) |   version: %s | go_version: %s | pid: %d
|_| |_| |_|\__,_|\__, |_| |_|\__, |      \__, |\___/ 
                 |___/       |___/       |___/       `, config.Version, buildInfo.GoVersion, os.Getpid()))
}
