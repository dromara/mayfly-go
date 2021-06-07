package apis

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/utils"
	"mayfly-go/server/sys/apis/form"
	"mayfly-go/server/sys/apis/vo"
	"mayfly-go/server/sys/application"
	"mayfly-go/server/sys/domain/entity"
	"strconv"
	"strings"
	"time"
)

type Account struct {
	AccountApp  application.IAccount
	ResourceApp application.IResource
	RoleApp     application.IRole
}

// @router /accounts/login [post]
func (a *Account) Login(rc *ctx.ReqCtx) {
	loginForm := &form.LoginForm{}
	ginx.BindJsonAndValid(rc.GinCtx, loginForm)
	rc.ReqParam = loginForm.Username

	account := &entity.Account{Username: loginForm.Username, Password: loginForm.Password}
	biz.ErrIsNil(a.AccountApp.GetAccount(account, "Id", "Username", "Password", "Status"), "用户名或密码错误")
	biz.IsTrue(account.IsEnable(), "该账号不可用")

	var resources vo.AccountResourceVOList
	// 获取账号菜单资源
	a.ResourceApp.GetAccountResources(account.Id, &resources)
	// 菜单树与权限code数组
	var menus vo.AccountResourceVOList
	var permissions []string
	for _, v := range resources {
		if v.Type == entity.ResourceTypeMenu {
			menus = append(menus, v)
		} else {
			permissions = append(permissions, *v.Code)
		}
	}

	rc.ResData = map[string]interface{}{
		"token":       ctx.CreateToken(account.Id, account.Username),
		"username":    account.Username,
		"menus":       menus.ToTrees(0),
		"permissions": permissions,
	}
}

// @router /accounts [get]
func (a *Account) Accounts(rc *ctx.ReqCtx) {
	condition := &entity.Account{}
	rc.ResData = a.AccountApp.GetPageList(condition, ginx.GetPageParam(rc.GinCtx), new([]vo.AccountManageVO))
}

// 获取账号角色id列表，用户回显角色分配
func (a *Account) AccountRoleIds(rc *ctx.ReqCtx) {
	rc.ResData = a.RoleApp.GetAccountRoleIds(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

// 获取账号角色id列表，用户回显角色分配
func (a *Account) AccountRoles(rc *ctx.ReqCtx) {
	vos := new([]vo.AccountRoleVO)
	a.RoleApp.GetAccountRoles(uint64(ginx.PathParamInt(rc.GinCtx, "id")), vos)
	rc.ResData = vos
}

func (a *Account) AccountResources(rc *ctx.ReqCtx) {
	var resources vo.ResourceManageVOList
	// 获取账号菜单资源
	a.ResourceApp.GetAccountResources(uint64(ginx.PathParamInt(rc.GinCtx, "id")), &resources)
	rc.ResData = resources.ToTrees(0)
}

// 保存账号角色信息
func (a *Account) SaveRoles(rc *ctx.ReqCtx) {
	g := rc.GinCtx

	var form form.AccountRoleForm
	ginx.BindJsonAndValid(g, &form)
	aid := uint64(form.Id)
	rc.ReqParam = form

	// 将,拼接的字符串进行切割
	idsStr := strings.Split(form.RoleIds, ",")
	var newIds []interface{}
	for _, v := range idsStr {
		id, _ := strconv.Atoi(v)
		newIds = append(newIds, uint64(id))
	}

	// 将[]uint64转为[]interface{}
	oIds := a.RoleApp.GetAccountRoleIds(uint64(form.Id))
	var oldIds []interface{}
	for _, v := range oIds {
		oldIds = append(oldIds, v)
	}

	addIds, delIds, _ := utils.ArrayCompare(newIds, oldIds, func(i1, i2 interface{}) bool {
		return i1.(uint64) == i2.(uint64)
	})

	createTime := time.Now()
	creator := rc.LoginAccount.Username
	creatorId := rc.LoginAccount.Id
	for _, v := range addIds {
		rr := &entity.AccountRole{AccountId: aid, RoleId: v.(uint64), CreateTime: &createTime, CreatorId: creatorId, Creator: creator}
		a.RoleApp.SaveAccountRole(rr)
	}
	for _, v := range delIds {
		a.RoleApp.DeleteAccountRole(aid, v.(uint64))
	}
}
