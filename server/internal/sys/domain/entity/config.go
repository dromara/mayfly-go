package entity

import (
	"encoding/json"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
	"strconv"
)

const (
	ConfigKeyAccountLoginSecurity string = "AccountLoginSecurity" // 账号登录安全配置
	ConfigKeyOauth2Login          string = "Oauth2Login"          // oauth2认证登录配置
	ConfigKeyLdapLogin            string = "LdapLogin"            // ldap登录配置
	ConfigKeyDbQueryMaxCount      string = "DbQueryMaxCount"      // 数据库查询的最大数量
	ConfigKeyDbSaveQuerySQL       string = "DbSaveQuerySQL"       // 数据库是否记录查询相关sql
	ConfigUseWartermark           string = "UseWartermark"        // 是否使用水印
)

type Config struct {
	model.Model
	Name       string `json:"name"` // 配置名
	Key        string `json:"key"`  // 配置key
	Params     string `json:"params" gorm:"column:params;type:varchar(1500)"`
	Value      string `json:"value" gorm:"column:value;type:varchar(1500)"`
	Remark     string `json:"remark"`
	Permission string `json:"permission"` // 可操作该配置的权限
}

func (a *Config) TableName() string {
	return "t_sys_config"
}

// 若配置信息不存在, 则返回传递的默认值.
func (c *Config) BoolValue(defaultValue bool) bool {
	// 如果值不存在，则返回默认值
	if c.Id == 0 {
		return defaultValue
	}
	return c.ConvBool(c.Value, defaultValue)
}

// 值返回json map
func (c *Config) GetJsonMap() map[string]string {
	var res map[string]string
	if c.Id == 0 || c.Value == "" {
		return res
	}
	_ = json.Unmarshal([]byte(c.Value), &res)
	return res
}

// 获取配置的int值，如果配置值非int或不存在，则返回默认值
func (c *Config) IntValue(defaultValue int) int {
	// 如果值不存在，则返回默认值
	if c.Id == 0 {
		return defaultValue
	}
	return c.ConvInt(c.Value, defaultValue)
}

type AccountLoginSecurity struct {
	UseCaptcha     bool   // 是否使用登录验证码
	UseOtp         bool   // 是否双因素校验
	OtpIssuer      string // otp发行人
	LoginFailCount int    // 允许失败次数
	LoginFailMin   int    // 登录失败指定次数后禁止的分钟数
}

// 转换为AccountLoginSecurity结构体
func (c *Config) ToAccountLoginSecurity() *AccountLoginSecurity {
	jm := c.GetJsonMap()
	als := new(AccountLoginSecurity)
	als.UseCaptcha = c.ConvBool(jm["useCaptcha"], true)
	als.UseOtp = c.ConvBool(jm["useOtp"], false)
	als.LoginFailCount = c.ConvInt(jm["loginFailCount"], 5)
	als.LoginFailMin = c.ConvInt(jm["loginFailMin"], 10)
	otpIssuer := jm["otpIssuer"]
	if otpIssuer == "" {
		otpIssuer = "mayfly-go"
	}
	als.OtpIssuer = otpIssuer
	return als
}

type ConfigOauth2Login struct {
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

// 转换为Oauth2Login结构体
func (c *Config) ToOauth2Login() *ConfigOauth2Login {
	jm := c.GetJsonMap()
	ol := new(ConfigOauth2Login)
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

type ConfigLdapLogin struct {
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

// 转换为LdapLogin结构体
func (c *Config) ToLdapLogin() *ConfigLdapLogin {
	jm := c.GetJsonMap()
	ll := new(ConfigLdapLogin)
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

// 转换配置中的值为bool类型（默认"1"或"true"为true，其他为false）
func (c *Config) ConvBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	return value == "1" || value == "true"
}

// 转换配置值中的值为int
func (c *Config) ConvInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	if intV, err := strconv.Atoi(value); err != nil {
		return defaultValue
	} else {
		return intV
	}
}
