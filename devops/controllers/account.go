package controllers

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/model"
	"mayfly-go/devops/controllers/form"
	"mayfly-go/devops/models"
)

// @router /accounts/login [post]
func Login(rc *ctx.ReqCtx) {
	loginForm := &form.LoginForm{}
	ginx.BindJsonAndValid(rc.GinCtx, loginForm)
	rc.ReqParam = loginForm.Username

	a := &models.Account{Username: loginForm.Username, Password: loginForm.Password}
	biz.BizErrIsNil(model.GetBy(a, "Id", "Username", "Password"), "用户名或密码错误")
	rc.ResData = map[string]interface{}{
		"token":    ctx.CreateToken(a.Id, a.Username),
		"username": a.Username,
	}
}

// @router /accounts [get]
func Accounts(rc *ctx.ReqCtx) {
	rc.ResData = models.ListAccount(ginx.GetPageParam(rc.GinCtx))
}
