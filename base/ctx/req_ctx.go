package ctx

import (
	"mayfly-go/base/ginx"
	"mayfly-go/base/model"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 处理函数
type HandlerFunc func(*ReqCtx)

type ReqCtx struct {
	Req    *http.Request // http request
	GinCtx *gin.Context  // gin context

	NeedToken    bool                // 是否需要token
	LoginAccount *model.LoginAccount // 登录账号信息

	LogInfo  *LogInfo    // 日志相关信息
	ReqParam interface{} // 请求参数，主要用于记录日志
	ResData  interface{} // 响应结果
	err      interface{} // 请求错误
	timed    int64       // 执行时间
}

func (rc *ReqCtx) Handle(handler HandlerFunc) {
	ginCtx := rc.GinCtx
	defer func() {
		if err := recover(); err != nil {
			rc.err = err
			ginx.ErrorRes(ginCtx, err)
		}
		// 应用所有请求后置处理器
		ApplyAfterHandler(rc)
	}()
	if ginCtx == nil {
		panic("ginContext == nil")
	}

	rc.Req = ginCtx.Request
	// 默认为不记录请求参数，可在handler回调函数中覆盖赋值
	rc.ReqParam = 0
	// 默认响应结果为nil，可在handler中赋值
	rc.ResData = nil

	// 调用请求前所有处理器
	err := ApplyBeforeHandler(rc)
	if err != nil {
		panic(err)
	}

	begin := time.Now()
	handler(rc)
	rc.timed = time.Now().Sub(begin).Milliseconds()
	ginx.SuccessRes(ginCtx, rc.ResData)
}

// 请求前置处理器，返回error则停止后续逻辑
type BeforeHandler interface {
	BeforeHandle(rc *ReqCtx) error
}

// 请求后置处理器
type AfterHandler interface {
	AfterHandle(rc *ReqCtx)
}

var (
	BeforeHandlers []BeforeHandler
	AfterHandlers  []AfterHandler
)

// 应用所有请求前置处理器
func ApplyBeforeHandler(rc *ReqCtx) error {
	for _, e := range BeforeHandlers {
		if err := e.BeforeHandle(rc); err != nil {
			return err
		}
	}
	return nil
}

// 应用所有后置处理器
func ApplyAfterHandler(rc *ReqCtx) {
	for _, e := range AfterHandlers {
		e.AfterHandle(rc)
	}
}

// 新建请求上下文，默认需要校验token
func NewReqCtx() *ReqCtx {
	return &ReqCtx{NeedToken: true}
}

func NewReqCtxWithGin(g *gin.Context) *ReqCtx {
	return &ReqCtx{NeedToken: true, GinCtx: g}
}

// 调用该方法设置请求描述，则默认记录日志，并不记录响应结果
func (r *ReqCtx) WithLog(li *LogInfo) *ReqCtx {
	r.LogInfo = li
	return r
}

// 是否需要token
func (r *ReqCtx) WithNeedToken(needToken bool) *ReqCtx {
	r.NeedToken = needToken
	return r
}
