package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogAccountLogin:   "Account Login",
	LogOauth2Callback: "Oauth2 Callback",
	LogOauth2Unbind:   "Oauth2 Unbind",
	LogLdapLogin:      "LDAP Login",

	ErrCaptchaErr:           "Captcha error",
	ErrLoginRestrict:        "login failed more than {{.failCount}} times, try again in {{.min}} minutes",
	ErrLoginFail:            "Wrong username or password [Login failed {{.failCount}} times]",
	ErrOtpTokenInvalid:      "otpToken error or invalid, please login again to obtain",
	ErrOtpCheckRestrict:     "Two-factor validation failed more than 5 times. Try again in 10 minutes",
	ErrOtpCheckFail:         "Two-factor authentication authorization code is incorrect",
	ErrAccountNotAvailable:  "Account is not available",
	LoginMsg:                "Log in to [{{.ip}}]-[{{.time}}]",
	ErrUsernameOrPwdErr:     "Wrong username or password",
	ErrOauth2NoAutoRegister: "the system does not enable automatic registration, please ask the administrator to add the corresponding account first",
}
