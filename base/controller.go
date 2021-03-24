package base

import (
	"encoding/json"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/mlog"
	"mayfly-go/base/model"

	"github.com/beego/beego/v2/core/validation"
	"github.com/beego/beego/v2/server/web"
)

type Controller struct {
	web.Controller
}

// 获取数据函数
type getDataFunc func(loginAccount *ctx.LoginAccount) interface{}

// 操作函数，无返回数据
type operationFunc func(loginAccount *ctx.LoginAccount)

// 将请求体的json赋值给指定的结构体
func (c *Controller) UnmarshalBody(data interface{}) {
	err := json.Unmarshal(c.Ctx.Input.RequestBody, data)
	biz.BizErrIsNil(err, "request body解析错误")
}

// 校验表单数据
func (c *Controller) validForm(form interface{}) {
	valid := validation.Validation{}
	b, err := valid.Valid(form)
	if err != nil {
		panic(err)
	}
	if !b {
		e := valid.Errors[0]
		panic(biz.NewBizErr(e.Field + " " + e.Message))
	}
}

// 将请求体的json赋值给指定的结构体，并校验表单数据
func (c *Controller) UnmarshalBodyAndValid(data interface{}) {
	c.UnmarshalBody(data)
	c.validForm(data)
}

// 返回数据
// @param reqCtx  请求上下文
// @param getData  获取数据的回调函数
func (c *Controller) ReturnData(reqCtx *ctx.ReqCtx, getData getDataFunc) {
	defer func() {
		if err := recover(); err != nil {
			ctx.ApplyAfterHandler(reqCtx, err.(error))
			c.parseErr(err)
		} else {
			// 应用所有请求后置处理器
			ctx.ApplyAfterHandler(reqCtx, nil)
		}
	}()
	reqCtx.Req = c.Ctx.Request
	// 调用请求前所有处理器
	err := ctx.ApplyBeforeHandler(reqCtx)
	if err != nil {
		panic(err)
	}

	resp := getData(reqCtx.LoginAccount)
	c.Success(resp)
	reqCtx.RespObj = resp
}

// 无返回数据的操作，如新增修改等无需返回数据的操作
// @param reqCtx  请求上下文
func (c *Controller) Operation(reqCtx *ctx.ReqCtx, operation operationFunc) {
	defer func() {
		if err := recover(); err != nil {
			ctx.ApplyAfterHandler(reqCtx, err.(error))
			c.parseErr(err)
		} else {
			ctx.ApplyAfterHandler(reqCtx, nil)
		}
	}()
	reqCtx.Req = c.Ctx.Request
	// 调用请求前所有处理器
	err := ctx.ApplyBeforeHandler(reqCtx)
	if err != nil {
		panic(err)
	}

	operation(reqCtx.LoginAccount)
	c.SuccessNoData()

	// 不记录返回结果
	reqCtx.RespObj = 0
}

// 获取分页参数
func (c *Controller) GetPageParam() *model.PageParam {
	pn, err := c.GetInt("pageNum", 1)
	biz.BizErrIsNil(err, "pageNum参数错误")
	ps, serr := c.GetInt("pageSize", 10)
	biz.BizErrIsNil(serr, "pageSize参数错误")
	return &model.PageParam{PageNum: pn, PageSize: ps}
}

// 统一返回Result json对象
func (c *Controller) Result(result *model.Result) {
	c.Data["json"] = result
	c.ServeJSON()
}

// 返回成功结果
func (c *Controller) Success(data interface{}) {
	c.Result(model.Success(data))
}

// 返回成功结果
func (c *Controller) SuccessNoData() {
	c.Result(model.SuccessNoData())
}

// 返回业务错误
func (c *Controller) BizError(bizError *biz.BizError) {
	c.Result(model.Error(bizError.Code(), bizError.Error()))
}

// 返回服务器错误结果
func (c *Controller) ServerError() {
	c.Result(model.ServerError())
}

// 解析error，并对不同error返回不同result
func (c *Controller) parseErr(err interface{}) {
	switch t := err.(type) {
	case *biz.BizError:
		c.BizError(t)
		break
	case error:
		c.ServerError()
		mlog.Log.Error(t)
		panic(err)
		//break
	case string:
		c.ServerError()
		mlog.Log.Error(t)
		panic(err)
		//break
	default:
		mlog.Log.Error(t)
	}
}
