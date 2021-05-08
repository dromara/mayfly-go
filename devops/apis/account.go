package apis

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/devops/apis/form"
	"mayfly-go/devops/application"
	"mayfly-go/devops/domain/entity"
	"mayfly-go/devops/models"
)

type Account struct {
	AccountApp application.IAccount
}

// @router /accounts/login [post]
func (a *Account) Login(rc *ctx.ReqCtx) {
	loginForm := &form.LoginForm{}
	ginx.BindJsonAndValid(rc.GinCtx, loginForm)
	rc.ReqParam = loginForm.Username

	account := &entity.Account{Username: loginForm.Username, Password: loginForm.Password}
	biz.ErrIsNil(a.AccountApp.GetAccount(account, "Id", "Username", "Password"), "用户名或密码错误")
	rc.ResData = map[string]interface{}{
		"token":    ctx.CreateToken(account.Id, account.Username),
		"username": account.Username,
	}
}

// @router /accounts [get]
func (a *Account) Accounts(rc *ctx.ReqCtx) {
	rc.ResData = models.ListAccount(ginx.GetPageParam(rc.GinCtx))
}
