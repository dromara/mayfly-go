package apis

import (
	"fmt"
	"mayfly-go/base/biz"
	"mayfly-go/base/captcha"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/global"
	"mayfly-go/base/httpclient"
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
	AccountApp  application.Account
	ResourceApp application.Resource
	RoleApp     application.Role
}

// @router /accounts/login [post]
func (a *Account) Login(rc *ctx.ReqCtx) {
	loginForm := &form.LoginForm{}
	ginx.BindJsonAndValid(rc.GinCtx, loginForm)
	rc.ReqParam = loginForm.Username

	// 校验验证码
	biz.IsTrue(captcha.Verify(loginForm.Cid, loginForm.Captcha), "验证码错误")

	account := &entity.Account{Username: loginForm.Username, Password: utils.Md5(loginForm.Password)}
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
	// 保存该账号的权限codes
	ctx.SavePermissionCodes(account.Id, permissions)

	// 保存登录消息
	go a.saveLogin(account, rc.GinCtx.ClientIP())

	rc.ResData = map[string]interface{}{
		"token":       ctx.CreateToken(account.Id, account.Username),
		"username":    account.Username,
		"menus":       menus.ToTrees(0),
		"permissions": permissions,
	}
}

// 保存更新账号登录信息
func (a *Account) saveLogin(account *entity.Account, ip string) {
	// 更新账号最后登录时间
	now := time.Now()
	updateAccount := &entity.Account{LastLoginTime: &now}
	updateAccount.Id = account.Id
	a.AccountApp.Update(updateAccount)

	bodyMap, err := httpclient.NewRequest(fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip)).Get().BodyToMap()
	if err != nil {
		global.Log.Errorf("获取客户端ip地址信息失败：%s", err.Error())
		return
	}
	if bodyMap["status"].(string) == "fail" {
		return
	}
	msg := fmt.Sprintf("%s-%s登录", bodyMap["regionName"], bodyMap["city"])
	fmt.Printf(msg)
}

// @router /accounts [get]
func (a *Account) Accounts(rc *ctx.ReqCtx) {
	condition := &entity.Account{}
	condition.Username = rc.GinCtx.Query("username")
	rc.ResData = a.AccountApp.GetPageList(condition, ginx.GetPageParam(rc.GinCtx), new([]vo.AccountManageVO))
}

// @router /accounts [get]
func (a *Account) CreateAccount(rc *ctx.ReqCtx) {
	form := &form.AccountCreateForm{}
	ginx.BindJsonAndValid(rc.GinCtx, form)
	rc.ReqParam = form

	account := &entity.Account{}
	utils.Copy(account, form)
	account.SetBaseInfo(rc.LoginAccount)
	a.AccountApp.Create(account)
}

func (a *Account) ChangeStatus(rc *ctx.ReqCtx) {
	g := rc.GinCtx

	account := &entity.Account{}
	account.Id = uint64(ginx.PathParamInt(g, "id"))
	account.Status = int8(ginx.PathParamInt(g, "status"))
	rc.ReqParam = fmt.Sprintf("accountId: %d, status: %d", account.Id, account.Status)
	a.AccountApp.Update(account)
}

func (a *Account) DeleteAccount(rc *ctx.ReqCtx) {
	id := uint64(ginx.PathParamInt(rc.GinCtx, "id"))
	rc.ReqParam = id
	a.AccountApp.Delete(id)
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
