package controllers

import (
	"mayfly-go/base"
	"mayfly-go/controllers/form"
	"mayfly-go/models"
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
	c.ReturnData(false, func(la *base.LoginAccount) interface{} {
		loginForm := &form.LoginForm{}
		c.UnmarshalBodyAndValid(loginForm)

		a := &models.Account{Username: loginForm.Username, Password: loginForm.Password}
		base.BizErrIsNil(base.GetBy(a, "Username", "Password"), "用户名或密码错误")
		return map[string]interface{}{
			"token":    base.CreateToken(a.Id, a.Username),
			"username": a.Username,
		}
	})
}

// @router /accounts [get]
func (c *AccountController) Accounts() {
	c.ReturnData(true, func(account *base.LoginAccount) interface{} {
		//s := c.GetString("username")
		//query := models.QuerySetter(new(models.Account)).OrderBy("-Id").RelatedSel()
		//return models.GetPage(query, c.GetPageParam(), new([]models.Account), new([]vo.AccountVO))

		return models.ListAccount(c.GetPageParam())
	})
}
