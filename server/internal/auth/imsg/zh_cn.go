package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogAccountLogin:   "账号登录",
	LogOauth2Callback: "Oauth2回调",
	LogOauth2Unbind:   "Oauth2解绑",
	LogLdapLogin:      "LDAP登录",

	ErrCaptchaErr:           "验证码错误",
	ErrLoginRestrict:        "登录失败超过{{.failCount}}次, 请{{.min}}分钟后再试",
	ErrLoginFail:            "用户名或密码错误【当前登录失败{{.failCount}}次】",
	ErrOtpTokenInvalid:      "otpToken错误或失效, 请重新登陆获取",
	ErrOtpCheckRestrict:     "双因素校验失败超过5次, 请10分钟后再试",
	ErrOtpCheckFail:         "双因素认证授权码不正确",
	ErrAccountNotAvailable:  "账号不可用",
	LoginMsg:                "于[{{.ip}}]-[{{.time}}]登录",
	ErrUsernameOrPwdErr:     "用户名或密码错误",
	ErrOauth2NoAutoRegister: "系统未开启自动注册, 请先让管理员添加对应账号",
}
