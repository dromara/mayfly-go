package config

import (
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/utils/conv"
	"mayfly-go/pkg/utils/stringx"
)

const (
	ConfigKeyAccountLoginSecurity string = "AccountLoginSecurity" // 账号登录安全配置
	ConfigKeyOauth2Login          string = "Oauth2Login"          // oauth2认证登录配置
	ConfigKeyLdapLogin            string = "LdapLogin"            // ldap登录配置
)

type AccountLoginSecurity struct {
	UseCaptcha     bool   // 是否使用登录验证码
	UseOtp         bool   // 是否双因素校验
	OtpIssuer      string // otp发行人
	LoginFailCount int    // 允许失败次数
	LoginFailMin   int    // 登录失败指定次数后禁止的分钟数
}

// 获取账号登录安全相关配置
func GetAccountLoginSecurity() *AccountLoginSecurity {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyAccountLoginSecurity)
	jm := c.GetJsonMap()
	als := new(AccountLoginSecurity)
	als.UseCaptcha = c.ConvBool(jm["useCaptcha"], true)
	als.UseOtp = c.ConvBool(jm["useOtp"], false)
	als.LoginFailCount = conv.Str2Int(jm["loginFailCount"], 5)
	als.LoginFailMin = conv.Str2Int(jm["loginFailMin"], 10)
	otpIssuer := jm["otpIssuer"]
	if otpIssuer == "" {
		otpIssuer = "mayfly-go"
	}
	als.OtpIssuer = otpIssuer
	return als
}

type Oauth2Login struct {
	Enable           bool // 是否启用
	Name             string
	ClientId         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
	AuthorizationURL string `json:"authorizationURL"`
	AccessTokenURL   string `json:"accessTokenURL"`
	RedirectURL      string `json:"redirectURL"`
	Scopes           string `json:"scopes"`
	ResourceURL      string `json:"resourceURL"`
	UserIdentifier   string `json:"userIdentifier"`
	AutoRegister     bool   `json:"autoRegister"` // 是否自动注册
}

// 获取Oauth2登录相关配置
func GetOauth2Login() *Oauth2Login {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyOauth2Login)
	jm := c.GetJsonMap()
	ol := new(Oauth2Login)
	ol.Enable = c.ConvBool(jm["enable"], false)
	ol.Name = jm["name"]
	ol.ClientId = jm["clientId"]
	ol.ClientSecret = jm["clientSecret"]
	ol.AuthorizationURL = jm["authorizationURL"]
	ol.AccessTokenURL = jm["accessTokenURL"]
	ol.RedirectURL = jm["redirectURL"]
	ol.Scopes = stringx.Trim(jm["scopes"])
	ol.ResourceURL = jm["resourceURL"]
	ol.UserIdentifier = jm["userIdentifier"]
	ol.AutoRegister = c.ConvBool(jm["autoRegister"], true)
	return ol
}

type LdapLogin struct {
	Enable           bool // 是否启用
	Host             string
	Port             string `json:"port"`
	SkipTLSVerify    bool   `json:"skipTLSVerify"`    // 客户端是否跳过 TLS 证书验证
	SecurityProtocol string `json:"securityProtocol"` // 安全协议（为Null不使用安全协议），如: StartTLS, LDAPS
	BindDN           string `json:"bindDn"`           // LDAP 服务的管理员账号，如: "cn=admin,dc=example,dc=com"
	BindPwd          string `json:"bindPwd"`          // LDAP 服务的管理员密码
	BaseDN           string `json:"baseDN"`           // 用户所在的 base DN, 如: "ou=users,dc=example,dc=com"
	UserFilter       string `json:"userFilter"`       // 过滤用户的方式, 如: "(uid=%s)"
	UidMap           string `json:"UidMap"`           // 用户id和 LDAP 字段名之间的映射关系
	UdnMap           string `json:"UdnMap"`           // 用户姓名(dispalyName)和 LDAP 字段名之间的映射关系
	EmailMap         string `json:"emailMap"`         // 用户email和 LDAP 字段名之间的映射关系
}

// 获取LdapLogin相关配置
func GetLdapLogin() *LdapLogin {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyLdapLogin)
	jm := c.GetJsonMap()
	ll := new(LdapLogin)
	ll.Enable = c.ConvBool(jm["enable"], false)
	ll.Host = jm["host"]
	ll.Port = jm["port"]
	ll.SkipTLSVerify = c.ConvBool(jm["skipTLSVerify"], true)
	ll.SecurityProtocol = jm["securityProtocol"]
	ll.BindDN = stringx.Trim(jm["bindDN"])
	ll.BindPwd = stringx.Trim(jm["bindPwd"])
	ll.BaseDN = stringx.Trim(jm["baseDN"])
	ll.UserFilter = stringx.Trim(jm["userFilter"])
	ll.UidMap = stringx.Trim(jm["uidMap"])
	ll.UdnMap = stringx.Trim(jm["udnMap"])
	ll.EmailMap = stringx.Trim(jm["emailMap"])
	return ll
}
