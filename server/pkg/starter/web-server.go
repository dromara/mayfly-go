package starter

import (
	"context"
	"errors"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/middleware"
	"mayfly-go/pkg/req"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func runWebServer(ctx context.Context, serverConfig ServerConf, options *Options) error {
	// 设置gin日志输出器
	logOut := logx.GetConfig().GetLogOut()
	gin.DefaultErrorWriter = logOut
	gin.DefaultWriter = logOut

	gin.SetMode(serverConfig.Model)

	// i18n配置
	i18n.SetLang(serverConfig.Lang)

	var router = gin.New()
	router.MaxMultipartMemory = 8 << 20
	// 初始化接口路由
	initRouter(router, req.RouterConfig{ContextPath: serverConfig.ContextPath})
	// 设置静态资源
	setStatic(router, serverConfig, options.StaticRouter)
	if options != nil && options.OnRoutesReady != nil {
		options.OnRoutesReady(router)
	}

	// 是否允许跨域
	if serverConfig.Cors {
		router.Use(middleware.Cors())
	}

	srv := http.Server{
		Addr: serverConfig.GetPort(),
		// 注册路由
		Handler: router,
	}

	go func() {
		defer gox.Recover()
		<-ctx.Done()
		logx.Info("Shutdown HTTP Server ...")
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(timeout)
		if err != nil {
			logx.Errorf("Failed to Shutdown HTTP Server: %v", err)
		}

		Terminate()
	}()

	logx.Infof("Listening and serving HTTP on %s", srv.Addr+serverConfig.ContextPath)
	var err error
	if serverConfig.TLS.Enable {
		err = srv.ListenAndServeTLS(serverConfig.TLS.CertFile, serverConfig.TLS.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}

	if errors.Is(err, http.ErrServerClosed) {
		logx.Info("HTTP Server Shutdown")
		return nil
	}

	if err != nil {
		logx.Errorf("Failed to Start HTTP Server: %v", err)
	}

	return err
}

func setStatic(router *gin.Engine, serverConfig ServerConf, staticRouter *StaticRouter) {
	contextPath := serverConfig.ContextPath

	if staticRouter != nil {
		fileServer := http.FileServer(http.FS(staticRouter.Fs))
		handler := WrapStaticHandler(http.StripPrefix(contextPath, fileServer))
		for _, p := range staticRouter.Paths {
			router.GET(contextPath+p, handler)
		}
	}

	// 设置静态资源
	for _, scs := range serverConfig.Statics {
		router.StaticFS(scs.RelativePath, http.Dir(scs.Root))
	}
	// 设置静态文件
	for _, sfs := range serverConfig.StaticFiles {
		router.StaticFile(sfs.RelativePath, sfs.Filepath)
	}
}

func WrapStaticHandler(h http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", `public, max-age=31536000`)
		h.ServeHTTP(c.Writer, c.Request)
	}
}
