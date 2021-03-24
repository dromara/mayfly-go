package ctx

import (
	"net/http"
)

type ReqCtx struct {
	Req          *http.Request
	NeedToken    bool          // 是否需要token
	LoginAccount *LoginAccount // 登录账号信息

	// 日志相关信息
	NeedLog     bool        // 是否需要记录日志
	LogResp     bool        // 是否记录返回结果
	Description string      // 请求描述
	ReqParam    interface{} // 请求参数
	RespObj     interface{} // 响应结果
}

// 请求前置处理器
type BeforeHandler interface {
	Handler(rc *ReqCtx) error
}

// 请求后置处理器
type AfterHandler interface {
	Handler(rc *ReqCtx, err error)
}

var (
	BeforeHandlers []BeforeHandler
	AfterHandlers  []AfterHandler
)

// 应用所有请求前置处理器
func ApplyBeforeHandler(rc *ReqCtx) error {
	for _, e := range BeforeHandlers {
		err := e.Handler(rc)
		if err != nil {
			return err
		}
	}
	return nil
}

// 应用所有后置处理器
func ApplyAfterHandler(rc *ReqCtx, err error) {
	for _, e := range AfterHandlers {
		e.Handler(rc, err)
	}
}

// 新建请求上下文，默认为需要记录日志
// @param needToken 是否需要token才可访问
// @param description 请求描述
func NewReqCtx(needToken bool, description string) *ReqCtx {
	return &ReqCtx{NeedToken: needToken, Description: description, NeedLog: true}
}

func NewNoLogReqCtx(needToken bool) *ReqCtx {
	return &ReqCtx{NeedToken: needToken, NeedLog: false}
}
