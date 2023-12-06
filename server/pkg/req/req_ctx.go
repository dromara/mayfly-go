package req

import (
	"context"
	"io"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/assert"
	"time"

	"github.com/gin-gonic/gin"
)

// 处理函数
type HandlerFunc func(*Ctx)

type Ctx struct {
	Conf *Conf // 请求配置

	GinCtx   *gin.Context // gin context
	ReqParam any          // 请求参数，主要用于记录日志
	LogExtra any          // 日志额外参数，主要用于系统日志定制化展示
	ResData  any          // 响应结果
	Err      any          // 请求错误
	timed    int64        // 执行时间

	MetaCtx context.Context // 元数据上下文信息，如登录账号(只有校验token后才会有值)，traceId等
}

func (rc *Ctx) Handle(handler HandlerFunc) {
	ginCtx := rc.GinCtx
	begin := time.Now()
	defer func() {
		rc.timed = time.Since(begin).Milliseconds()
		if err := recover(); err != nil {
			rc.Err = err
			ginx.ErrorRes(ginCtx, err)
		}
		// 应用所有请求后置处理器
		ApplyHandlerInterceptor(afterHandlers, rc)
	}()
	assert.IsTrue(ginCtx != nil, "ginContext == nil")

	// 默认为不记录请求参数，可在handler回调函数中覆盖赋值
	rc.ReqParam = nil
	// 默认响应结果为nil，可在handler中赋值
	rc.ResData = nil

	// 调用请求前所有处理器
	err := ApplyHandlerInterceptor(beforeHandlers, rc)
	if err != nil {
		panic(err)
	}

	handler(rc)
	if rc.Conf == nil || !rc.Conf.noRes {
		ginx.SuccessRes(ginCtx, rc.ResData)
	}
}

func (rc *Ctx) Download(reader io.Reader, filename string) {
	ginx.Download(rc.GinCtx, reader, filename)
}

// 获取当前登录账号信息，不存在则会报错。
//
// 若不需要报错，则使用contextx.GetLoginAccount方法
func (rc *Ctx) GetLoginAccount() *model.LoginAccount {
	la := contextx.GetLoginAccount(rc.MetaCtx)
	biz.IsTrue(la != nil, "获取登录账号信息失败, 请确认该接口是否通过鉴权")
	return la
}

func (rc *Ctx) WithConf(conf *Conf) *Ctx {
	rc.Conf = conf
	return rc
}

// 设置请求上下文需要的权限信息
func (rc *Ctx) WithRequiredPermission(permission *Permission) *Ctx {
	if rc.Conf == nil {
		rc.Conf = new(Conf)
	}
	rc.Conf.requiredPermission = permission
	return rc
}

// 设置请求日志信息
func (rc *Ctx) WithLog(logInfo *LogInfo) *Ctx {
	if rc.Conf == nil {
		rc.Conf = new(Conf)
	}
	rc.Conf.logInfo = logInfo
	return rc
}

func (rc *Ctx) GetLogInfo() *LogInfo {
	return rc.Conf.logInfo
}

func NewCtxWithGin(g *gin.Context) *Ctx {
	return &Ctx{GinCtx: g, MetaCtx: contextx.WithTraceId(g.Request.Context())}
}

// 处理器拦截器函数
type HandlerInterceptorFunc func(*Ctx) error
type HandlerInterceptors []HandlerInterceptorFunc

var (
	beforeHandlers HandlerInterceptors
	afterHandlers  HandlerInterceptors
)

// 使用前置处理器函数
func UseBeforeHandlerInterceptor(b HandlerInterceptorFunc) {
	beforeHandlers = append(beforeHandlers, b)
}

// 使用后置处理器函数
func UseAfterHandlerInterceptor(b HandlerInterceptorFunc) {
	afterHandlers = append(afterHandlers, b)
}

// 应用指定处理器拦截器，如果有一个错误则直接返回错误
func ApplyHandlerInterceptor(his HandlerInterceptors, rc *Ctx) any {
	for _, handler := range his {
		if err := handler(rc); err != nil {
			return err
		}
	}
	return nil
}
