package starter

import (
	"context"
	"errors"
	"io/fs"
	"mayfly-go/initialize"
	"mayfly-go/internal/pkg/config"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/middleware"
	"mayfly-go/pkg/req"
	"mayfly-go/static"
	"net/http"
	"time"

	sysapp "mayfly-go/internal/sys/application"

	"github.com/gin-gonic/gin"
)

func runWebServer(ctx context.Context) {
	// 设置gin日志输出器
	logOut := logx.GetConfig().GetLogOut()
	gin.DefaultErrorWriter = logOut
	gin.DefaultWriter = logOut

	// 权限处理器
	req.UseBeforeHandlerInterceptor(req.PermissionHandler)
	// 日志处理器
	req.UseAfterHandlerInterceptor(req.LogHandler)
	// 设置日志保存函数
	req.SetSaveLogFunc(sysapp.GetSyslogApp().SaveFromReq)

	// jwt配置
	jwtConf := config.Conf.Jwt
	req.SetJwtConf(req.JwtConf{
		Key:                    jwtConf.Key,
		ExpireTime:             jwtConf.ExpireTime,
		RefreshTokenExpireTime: jwtConf.RefreshTokenExpireTime,
	})

	// i18n配置
	i18n.SetLang(config.Conf.Server.Lang)

	// server配置
	serverConfig := config.Conf.Server
	gin.SetMode(serverConfig.Model)

	var router = gin.New()
	router.MaxMultipartMemory = 8 << 20
	// 初始化接口路由
	initialize.InitRouter(router, initialize.RouterConfig{ContextPath: serverConfig.ContextPath})
	// 设置静态资源
	setStatic(serverConfig.ContextPath, router)
	// 是否允许跨域
	if serverConfig.Cors {
		router.Use(middleware.Cors())
	}

	srv := http.Server{
		Addr: config.Conf.Server.GetPort(),
		// 注册路由
		Handler: router,
	}

	go func() {
		defer gox.RecoverPanic()
		<-ctx.Done()
		logx.Info("Shutdown HTTP Server ...")
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(timeout)
		if err != nil {
			logx.Errorf("failed to Shutdown HTTP Server: %v", err)
		}

		initialize.Terminate()
	}()

	confSrv := config.Conf.Server
	logx.Infof("Listening and serving HTTP on %s", srv.Addr+confSrv.ContextPath)
	var err error
	if confSrv.Tls != nil && confSrv.Tls.Enable {
		err = srv.ListenAndServeTLS(confSrv.Tls.CertFile, confSrv.Tls.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if errors.Is(err, http.ErrServerClosed) {
		logx.Info("HTTP Server Shutdown")
	} else if err != nil {
		logx.Errorf("Failed to Start HTTP Server: %v", err)
	}
}

func setStatic(contextPath string, router *gin.Engine) {
	// 使用embed打包静态资源至二进制文件中
	fsys, _ := fs.Sub(static.Static, "static")
	fileServer := http.FileServer(http.FS(fsys))
	handler := WrapStaticHandler(http.StripPrefix(contextPath, fileServer))

	router.GET(contextPath+"/", handler)
	router.GET(contextPath+"/favicon.ico", handler)
	router.GET(contextPath+"/config.js", handler)
	// 所有/assets/**开头的都是静态资源文件
	router.GET(contextPath+"/assets/*file", handler)

	// 设置静态资源
	if staticConfs := config.Conf.Server.Static; staticConfs != nil {
		for _, scs := range *staticConfs {
			router.StaticFS(scs.RelativePath, http.Dir(scs.Root))
		}
	}
	// 设置静态文件
	if staticFileConfs := config.Conf.Server.StaticFile; staticFileConfs != nil {
		for _, sfs := range *staticFileConfs {
			router.StaticFile(sfs.RelativePath, sfs.Filepath)
		}
	}
}

func WrapStaticHandler(h http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", `public, max-age=31536000`)
		h.ServeHTTP(c.Writer, c.Request)
	}
}
