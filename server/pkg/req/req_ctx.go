package req

import (
	"cmp"
	"context"
	"fmt"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/assert"
	"net/http"
	"time"
)

// 处理函数
type HandlerFunc func(*Ctx)

type Ctx struct {
	*wrapperF       // http framework处理接口
	Conf      *Conf // 请求配置

	ReqParam any   // 请求参数，主要用于记录日志
	ResData  any   // 响应结果
	Error    any   // 请求错误
	timed    int64 // 执行时间

	MetaCtx context.Context // 元数据上下文信息，如登录账号(只有校验token后才会有值)，traceId等
}

func NewCtx(f F) *Ctx {
	metaCtx := contextx.WithTraceId(f.GetRequest().Context())
	metaCtx = i18n.NewCtxWithLang(metaCtx, f.GetRequest().Header.Get("Accept-Language"))

	ctx := &Ctx{MetaCtx: metaCtx}
	ctx.wrapperF = NewWrapperF(f)
	return ctx
}

// 执行指定handler func，并输出响应结果
func (rc *Ctx) Handle(handler HandlerFunc) {
	begin := time.Now()
	defer func() {
		rc.timed = time.Since(begin).Milliseconds()
		if err := recover(); err != nil {
			rc.Error = err
		}
		// 应用所有请求后置处理器
		ApplyHandlerInterceptor(afterHandlers, rc)
		// 输出响应结果
		rc.res()
	}()
	assert.IsTrue(rc.wrapperF != nil, "F == nil")

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

// 输出响应结果
func (rc *Ctx) res() {
	if err := rc.Error; err != nil {
		switch t := err.(type) {
		case *errorx.BizError:
			rc.JSONRes(http.StatusOK, model.Error(t))
		default:
			logx.ErrorTrace("server error", t)
			rc.JSONRes(http.StatusOK, model.ServerError(fmt.Sprintf("server error [%d-%s]", errorx.ServerError.Code(), cmp.Or(contextx.GetTraceId(rc.MetaCtx), "none"))))
		}
		return
	}

	if rc.Conf == nil || !rc.Conf.noRes {
		rc.JSONRes(http.StatusOK, model.Success(rc.ResData))
	}
}

/************************************/
/***** GOLANG.ORG/X/NET/CONTEXT -> copy gin.Context *****/
/************************************/

// hasRequestContext returns whether c.Request has Context and fallback.
func (c *Ctx) hasRequestContext() bool {
	request := c.GetRequest()
	return request != nil && request.Context() != nil
}

// Deadline returns that there is no deadline (ok==false) when c.Request has no Context.
func (c *Ctx) Deadline() (deadline time.Time, ok bool) {
	if !c.hasRequestContext() {
		return
	}
	return c.GetRequest().Context().Deadline()
}

// Done returns nil (chan which will wait forever) when c.Request has no Context.
func (c *Ctx) Done() <-chan struct{} {
	if !c.hasRequestContext() {
		return nil
	}
	return c.GetRequest().Context().Done()
}

// Err returns nil when c.Request has no Context.
func (c *Ctx) Err() error {
	if !c.hasRequestContext() {
		return nil
	}
	return c.GetRequest().Context().Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (c *Ctx) Value(key any) any {
	if key == 0 {
		return c.GetRequest()
	}
	if !c.hasRequestContext() {
		return nil
	}
	return c.GetRequest().Context().Value(key)
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
