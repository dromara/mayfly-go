package controllers

import (
	"mayfly-go/base"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/model"
	"mayfly-go/devops/controllers/form"
	"mayfly-go/devops/models"
)

type AccountController struct {
	base.Controller
}

//func (c *AccountController) URLMapping() {
//	c.Mapping("Login", c.Login)
//	c.Mapping("Accounts", c.Accounts)
//}

// @router /accounts/login [post]
func (c *AccountController) Login() {
	c.ReturnData(ctx.NewReqCtx(false, "用户登录"), func(la *ctx.LoginAccount) interface{} {
		loginForm := &form.LoginForm{}
		c.UnmarshalBodyAndValid(loginForm)

		a := &models.Account{Username: loginForm.Username, Password: loginForm.Password}
		biz.BizErrIsNil(model.GetBy(a, "Username", "Password"), "用户名或密码错误")
		return map[string]interface{}{
			"token":    ctx.CreateToken(a.Id, a.Username),
			"username": a.Username,
		}
	})
}

// @router /accounts [get]
func (c *AccountController) Accounts() {
	c.ReturnData(ctx.NewReqCtx(true, "获取账号列表"), func(account *ctx.LoginAccount) interface{} {
		//s := c.GetString("username")
		//query := models.QuerySetter(new(models.Account)).OrderBy("-Id").RelatedSel()
		//return models.GetPage(query, c.GetPageParam(), new([]models.Account), new([]vo.AccountVO))

		return models.ListAccount(c.GetPageParam())
	})
}
