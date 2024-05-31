package starter

import (
	"context"
	"errors"
	"mayfly-go/initialize"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
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

	srv := http.Server{
		Addr: config.Conf.Server.GetPort(),
		// 注册路由
		Handler: initialize.InitRouter(),
	}

	go func() {
		<-ctx.Done()
		logx.Info("Shutdown HTTP Server ...")
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(timeout)
		if err != nil {
			logx.Errorf("Failed to Shutdown HTTP Server: %v", err)
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
