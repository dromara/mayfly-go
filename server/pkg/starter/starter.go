package starter

import (
	"context"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/validatorx"
	"os"
	"os/signal"
	"syscall"
)

func Run(config Conf, opts ...Option) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	// 应用配置默认值
	if err := config.ApplyDefaults(); err != nil {
		return err
	}

	options := NewOptions(opts...)

	// 初始化并赋值数据库全局变量
	db, err := initDB(config.DB)
	if err != nil {
		return err
	}
	global.Db = db
	if options.OnDbReady != nil {
		if err := options.OnDbReady(db); err != nil {
			return err
		}
	}

	// 初始化缓存
	if err := initCache(config.Redis); err != nil {
		return err
	}

	// 初始化其他需要启动时运行的方法
	if err := initOther(); err != nil {
		return err
	}

	// 参数校验器初始化、如错误提示中文转译等
	validatorx.Init()

	// jwt配置
	jwtConf := config.Jwt
	req.SetJwtConf(req.JwtConf{
		Key:                    jwtConf.Key,
		ExpireTime:             jwtConf.ExpireTime,
		RefreshTokenExpireTime: jwtConf.RefreshTokenExpireTime,
	})

	// 权限处理器
	req.UseBeforeHandlerInterceptor(req.PermissionHandler)
	// 日志处理器
	req.UseAfterHandlerInterceptor(req.LogHandler)

	// 设置日志保存函数
	if options.LogSaver != nil {
		req.SetSaveLogFunc(options.LogSaver())
	}

	// 启动前回调
	if options.OnBeforeStart != nil {
		options.OnBeforeStart()
	}

	// 运行web服务
	return runWebServer(ctx, config.Server, options)
}
