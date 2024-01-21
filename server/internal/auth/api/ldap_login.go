package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"mayfly-go/internal/auth/api/form"
	"mayfly-go/internal/auth/config"
	msgapp "mayfly-go/internal/msg/application"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type LdapLogin struct {
	AccountApp sysapp.Account `inject:""`
	MsgApp     msgapp.Msg     `inject:""`
}

// @router /auth/ldap/enabled [get]
func (a *LdapLogin) GetLdapEnabled(rc *req.Ctx) {
	ldapLoginConfig := config.GetLdapLogin()
	rc.ResData = ldapLoginConfig.Enable
}

// @router /auth/ldap/login [post]
func (a *LdapLogin) Login(rc *req.Ctx) {
	loginForm := ginx.BindJsonAndValid(rc.GinCtx, new(form.LoginForm))

	accountLoginSecurity := config.GetAccountLoginSecurity()
	// 判断是否有开启登录验证码校验
	if accountLoginSecurity.UseCaptcha {
		// 校验验证码
		biz.IsTrue(captcha.Verify(loginForm.Cid, loginForm.Captcha), "验证码错误")
	}

	username := loginForm.Username

	clientIp := getIpAndRegion(rc)
	rc.ReqParam = collx.Kvs("username", username, "ip", clientIp)

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
		panic(errorx.NewBiz(fmt.Sprintf("用户名或密码错误【当前登录失败%d次】", nowFailCount)))
	}

	rc.ResData = LastLoginCheck(account, accountLoginSecurity, clientIp)
}

func (a *LdapLogin) getUser(userName string, cols ...string) (*sysentity.Account, error) {
	account := &sysentity.Account{Username: userName}
	if err := a.AccountApp.GetBy(account, cols...); err != nil {
		return nil, err
	}
	return account, nil
}

func (a *LdapLogin) createUser(userName, displayName string) {
	account := &sysentity.Account{Username: userName}
	account.FillBaseInfo(model.IdGenTypeNone, nil)
	account.Name = displayName
	biz.ErrIsNil(a.AccountApp.Create(context.TODO(), account))
	// 将 LADP 用户本地密码设置为空，不允许本地登录
	account.Password = cryptox.PwdHash("")
	biz.ErrIsNil(a.AccountApp.Update(context.TODO(), account))
}

func (a *LdapLogin) getOrCreateUserWithLdap(userName string, password string, cols ...string) (*sysentity.Account, error) {
	userInfo, err := Authenticate(userName, password)
	if err != nil {
		return nil, errors.New("用户名密码错误")
	}

	account, err := a.getUser(userName, cols...)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		a.createUser(userName, userInfo.DisplayName)
		return a.getUser(userName, cols...)
	} else if err != nil {
		return nil, err
	}
	return account, nil
}

type UserInfo struct {
	UserName    string
	DisplayName string
	Email       string
}

// Authenticate 通过 LDAP 验证用户名密码
func Authenticate(username, password string) (*UserInfo, error) {
	ldapConf := config.GetLdapLogin()
	if !ldapConf.Enable {
		return nil, errors.Errorf("未启用 LDAP 登录")
	}
	conn, err := Connect(ldapConf)
	if err != nil {
		return nil, errors.Errorf("connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	sr, err := conn.Search(
		ldap.NewSearchRequest(
			ldapConf.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0,
			0,
			false,
			strings.ReplaceAll(ldapConf.UserFilter, "%s", username),
			[]string{"dn", ldapConf.UidMap, ldapConf.UdnMap, ldapConf.EmailMap},
			nil,
		),
	)
	if err != nil {
		return nil, errors.Errorf("search user DN: %v", err)
	} else if len(sr.Entries) != 1 {
		return nil, errors.Errorf("expect 1 user DN but got %d", len(sr.Entries))
	}
	entry := sr.Entries[0]

	// Bind as the user to verify their password
	err = conn.Bind(entry.DN, password)
	if err != nil {
		return nil, errors.Errorf("bind user: %v", err)
	}

	userName := entry.GetAttributeValue(ldapConf.UidMap)
	if userName == "" {
		return nil, errors.Errorf("the attribute %q is not found or has empty value", ldapConf.UidMap)
	}
	return &UserInfo{
		UserName:    userName,
		DisplayName: entry.GetAttributeValue(ldapConf.UdnMap),
		Email:       entry.GetAttributeValue(ldapConf.EmailMap),
	}, nil
}

// Connect 创建 LDAP 连接
func Connect(ldapConf *config.LdapLogin) (*ldap.Conn, error) {
	conn, err := dial(ldapConf)
	if err != nil {
		return nil, err
	}

	// Bind with a system account
	err = conn.Bind(ldapConf.BindDN, ldapConf.BindPwd)
	if err != nil {
		_ = conn.Close()
		return nil, errors.Errorf("bind: %v", err)
	}
	return conn, nil
}

func dial(ldapConf *config.LdapLogin) (*ldap.Conn, error) {
	addr := fmt.Sprintf("%s:%s", ldapConf.Host, ldapConf.Port)
	tlsConfig := &tls.Config{
		ServerName:         ldapConf.Host,
		InsecureSkipVerify: ldapConf.SkipTLSVerify,
	}
	if ldapConf.SecurityProtocol == "LDAPS" {
		conn, err := ldap.DialTLS("tcp", addr, tlsConfig)
		if err != nil {
			return nil, errors.Errorf("dial TLS: %v", err)
		}
		return conn, nil
	}

	conn, err := ldap.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Errorf("dial: %v", err)
	}
	if ldapConf.SecurityProtocol == "StartTLS" {
		if err = conn.StartTLS(tlsConfig); err != nil {
			_ = conn.Close()
			return nil, errors.Errorf("start TLS: %v", err)
		}
	}
	return conn, nil
}
