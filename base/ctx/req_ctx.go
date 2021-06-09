package ctx

import (
	"mayfly-go/base/ginx"
	"mayfly-go/base/model"
	"time"

	"github.com/gin-gonic/gin"
)

// 处理函数
type HandlerFunc func(*ReqCtx)

type ReqCtx struct {
	GinCtx *gin.Context // gin context

	// NeedToken          bool                // 是否需要token
	RequiredPermission *Permission         // 需要的权限信息，默认为nil，需要校验token
	LoginAccount       *model.LoginAccount // 登录账号信息，只有校验token后才会有值

	LogInfo  *LogInfo    // 日志相关信息
	ReqParam interface{} // 请求参数，主要用于记录日志
	ResData  interface{} // 响应结果
	Err      interface{} // 请求错误

	timed int64 // 执行时间
	noRes bool  // 无需返回结果，即文件下载等
}

func (rc *ReqCtx) Handle(handler HandlerFunc) {
	ginCtx := rc.GinCtx
	defer func() {
		if err := recover(); err != nil {
			rc.Err = err
			ginx.ErrorRes(ginCtx, err)
		}
		// 应用所有请求后置处理器
		ApplyHandlerInterceptor(afterHandlers, rc)
	}()
	if ginCtx == nil {
		panic("ginContext == nil")
	}

	// 默认为不记录请求参数，可在handler回调函数中覆盖赋值
	rc.ReqParam = 0
	// 默认响应结果为nil，可在handler中赋值
	rc.ResData = nil

	// 调用请求前所有处理器
	err := ApplyHandlerInterceptor(beforeHandlers, rc)
	if err != nil {
		panic(err)
	}

	begin := time.Now()
	handler(rc)
	rc.timed = time.Now().Sub(begin).Milliseconds()
	if !rc.noRes {
		ginx.SuccessRes(ginCtx, rc.ResData)
	}
}

func (rc *ReqCtx) Download(data []byte, filename string) {
	rc.noRes = true
	ginx.Download(rc.GinCtx, data, filename)
}

// 新建请求上下文，默认需要校验token
func NewReqCtx() *ReqCtx {
	return &ReqCtx{}
}

func NewReqCtxWithGin(g *gin.Context) *ReqCtx {
	return &ReqCtx{GinCtx: g}
}

// 调用该方法设置请求描述，则默认记录日志，并不记录响应结果
func (r *ReqCtx) WithLog(li *LogInfo) *ReqCtx {
	r.LogInfo = li
	return r
}

// 设置请求上下文需要的权限信息
func (r *ReqCtx) WithRequiredPermission(permission *Permission) *ReqCtx {
	r.RequiredPermission = permission
	return r
}

// 是否需要token
func (r *ReqCtx) WithNeedToken(needToken bool) *ReqCtx {
	r.RequiredPermission = &Permission{NeedToken: false}
	return r
}

// 处理器拦截器函数
type HandlerInterceptorFunc func(*ReqCtx) error
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
func ApplyHandlerInterceptor(his HandlerInterceptors, rc *ReqCtx) interface{} {
	for _, handler := range his {
		if err := handler(rc); err != nil {
			return err
		}
	}
	return nil
}
