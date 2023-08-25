package api

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"mayfly-go/internal/auth/api/form"
	msgapp "mayfly-go/internal/msg/application"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/ldap"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/cryptox"
	"strconv"
	"time"
)

type LdapLogin struct {
	AccountApp sysapp.Account
	MsgApp     msgapp.Msg
	ConfigApp  sysapp.Config
}

// @router /auth/ldap/enabled [get]
func (a *LdapLogin) GetLdapEnabled(rc *req.Ctx) {
	rc.ResData = config.Conf.Ldap.Enabled
}

// @router /auth/ldap/login [post]
func (a *LdapLogin) Login(rc *req.Ctx) {
	loginForm := ginx.BindJsonAndValid(rc.GinCtx, new(form.LoginForm))

	// 确认是 LDAP 登录
	biz.IsTrue(loginForm.LdapLogin, "LDAP 登录参数错误")

	accountLoginSecurity := a.ConfigApp.GetConfig(sysentity.ConfigKeyAccountLoginSecurity).ToAccountLoginSecurity()
	// 判断是否有开启登录验证码校验
	if accountLoginSecurity.UseCaptcha {
		// 校验验证码
		biz.IsTrue(captcha.Verify(loginForm.Cid, loginForm.Captcha), "验证码错误")
	}

	username := loginForm.Username

	clientIp := getIpAndRegion(rc)
	rc.ReqParam = fmt.Sprintf("username: %s | ip: %s", username, clientIp)

	originPwd, err := cryptox.DefaultRsaDecrypt(loginForm.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	// LDAP 用户本地密码为空，不允许本地登录
	biz.NotEmpty(originPwd, "密码不能为空")

	failCountKey := fmt.Sprintf("account:login:failcount:%s", username)
	nowFailCount := cache.GetInt(failCountKey)
	loginFailCount := accountLoginSecurity.LoginFailCount
	loginFailMin := accountLoginSecurity.LoginFailMin
	biz.IsTrue(nowFailCount < loginFailCount, "登录失败超过%d次, 请%d分钟后再试", loginFailCount, loginFailMin)

	var account *sysentity.Account
	cols := []string{"Id", "Name", "Username", "Password", "Status", "LastLoginTime", "LastLoginIp", "OtpSecret"}
	account, err = a.getOrCreateUserWithLdap(username, originPwd, cols...)

	if err != nil {
		nowFailCount++
		cache.SetStr(failCountKey, strconv.Itoa(nowFailCount), time.Minute*time.Duration(loginFailMin))
		panic(biz.NewBizErr(fmt.Sprintf("用户名或密码错误【当前登录失败%d次】", nowFailCount)))
	}

	rc.ResData = LastLoginCheck(account, accountLoginSecurity, clientIp)
}

func (a *LdapLogin) getUser(userName string, cols ...string) (*sysentity.Account, error) {
	account := &sysentity.Account{Username: userName}
	if err := a.AccountApp.GetAccount(account, cols...); err != nil {
		return nil, err
	}
	return account, nil
}

func (a *LdapLogin) createUser(userName, displayName string) {
	account := &sysentity.Account{Username: userName}
	account.SetBaseInfo(nil)
	account.Name = displayName
	a.AccountApp.Create(account)
	// 将 LADP 用户本地密码设置为空，不允许本地登录
	account.Password = cryptox.PwdHash("")
	a.AccountApp.Update(account)
}

func (a *LdapLogin) getOrCreateUserWithLdap(userName string, password string, cols ...string) (*sysentity.Account, error) {
	userInfo, err := ldap.Authenticate(userName, password)
	if err != nil {
		return nil, errors.New("用户名密码错误")
	}

	account, err := a.getUser(userName, cols...)
	if err == gorm.ErrRecordNotFound {
		a.createUser(userName, userInfo.DisplayName)
		return a.getUser(userName, cols...)
	} else if err != nil {
		return nil, err
	}
	return account, nil
}
