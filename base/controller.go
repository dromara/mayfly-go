package base

import (
	"encoding/json"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/validation"
)

type Controller struct {
	beego.Controller
}

// 获取数据函数
type getDataFunc func(loginAccount *LoginAccount) interface{}

// 操作函数，无返回数据
type operationFunc func(loginAccount *LoginAccount)

// 将请求体的json赋值给指定的结构体
func (c *Controller) UnmarshalBody(data interface{}) {
	err := json.Unmarshal(c.Ctx.Input.RequestBody, data)
	BizErrIsNil(err, "request body解析错误")
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
		panic(NewBizErr(e.Field + " " + e.Message))
	}
}

// 将请求体的json赋值给指定的结构体，并校验表单数据
func (c *Controller) UnmarshalBodyAndValid(data interface{}) {
	c.UnmarshalBody(data)
	c.validForm(data)
}

// 返回数据
// @param checkToken  是否校验token
// @param getData  获取数据的回调函数
func (c *Controller) ReturnData(checkToken bool, getData getDataFunc) {
	defer func() {
		if err := recover(); err != nil {
			c.parseErr(err)
		}
	}()
	var loginAccount *LoginAccount
	if checkToken {
		loginAccount = c.CheckToken()
	}
	c.Success(getData(loginAccount))
}

// 无返回数据的操作，如新增修改等无需返回数据的操作
// @param checkToken  是否校验token
func (c *Controller) Operation(checkToken bool, operation operationFunc) {
	defer func() {
		if err := recover(); err != nil {
			c.parseErr(err)
		}
	}()
	var loginAccount *LoginAccount
	if checkToken {
		loginAccount = c.CheckToken()
	}
	operation(loginAccount)
	c.SuccessNoData()
}

// 校验token，并返回登录者账号信息
func (c *Controller) CheckToken() *LoginAccount {
	tokenStr := c.Ctx.Input.Header("Authorization")
	loginAccount, err := ParseToken(tokenStr)
	if err != nil || loginAccount == nil {
		panic(NewBizErrCode(TokenErrorCode, TokenErrorMsg))
	}
	return loginAccount
}

// 获取分页参数
func (c *Controller) GetPageParam() *PageParam {
	pn, err := c.GetInt("pageNum", 1)
	BizErrIsNil(err, "pageNum参数错误")
	ps, serr := c.GetInt("pageSize", 10)
	BizErrIsNil(serr, "pageSize参数错误")
	return &PageParam{PageNum: pn, PageSize: ps}
}

// 统一返回Result json对象
func (c *Controller) Result(result *Result) {
	c.Data["json"] = result
	c.ServeJSON()
}

// 返回成功结果
func (c *Controller) Success(data interface{}) {
	c.Result(Success(data))
}

// 返回成功结果
func (c *Controller) SuccessNoData() {
	c.Result(SuccessNoData())
}

// 返回业务错误
func (c *Controller) BizError(bizError BizError) {
	c.Result(Error(bizError.Code(), bizError.Error()))
}

// 返回服务器错误结果
func (c *Controller) ServerError() {
	c.Result(ServerError())
}

// 解析error，并对不同error返回不同result
func (c *Controller) parseErr(err interface{}) {
	switch t := err.(type) {
	case BizError:
		c.BizError(t)
		break
	case error:
		c.ServerError()
		logs.Error(t)
		panic(err)
		//break
	case string:
		c.ServerError()
		logs.Error(t)
		panic(err)
		//break
	default:
		logs.Error(t)
	}
}
