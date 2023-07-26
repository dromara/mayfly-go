package entity

import (
	"encoding/json"
	"mayfly-go/pkg/model"
	"strconv"
)

const (
	ConfigKeyAccountLoginSecurity string = "AccountLoginSecurity" // 账号登录安全配置
	ConfigKeyOauth2Login          string = "Oauth2Login"          // oauth2认证登录配置
	ConfigKeyDbQueryMaxCount      string = "DbQueryMaxCount"      // 数据库查询的最大数量
	ConfigKeyDbSaveQuerySQL       string = "DbSaveQuerySQL"       // 数据库是否记录查询相关sql
	ConfigUseWartermark           string = "UseWartermark"        // 是否使用水印
)

type Config struct {
	model.Model
	Name   string `json:"name"` // 配置名
	Key    string `json:"key"`  // 配置key
	Params string `json:"params" gorm:"column:params;type:varchar(1500)"`
	Value  string `json:"value" gorm:"column:value;type:varchar(1500)"`
	Remark string `json:"remark"`
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
	ol.Scopes = jm["scopes"]
	ol.ResourceURL = jm["resourceURL"]
	ol.UserIdentifier = jm["userIdentifier"]
	ol.AutoRegister = c.ConvBool(jm["autoRegister"], true)
	return ol
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
