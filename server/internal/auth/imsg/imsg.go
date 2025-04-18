package imsg

import (
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
)

func init() {
	i18n.AppendLangMsg(i18n.Zh_CN, Zh_CN)
	i18n.AppendLangMsg(i18n.En, En)
}

const (
	LogAccountLogin = iota + consts.ImsgNumAuth
	LogOauth2Callback
	LogOauth2Unbind
	LogLdapLogin

	ErrCaptchaErr
	ErrLoginRestrict
	ErrLoginFail
	ErrOtpTokenInvalid
	ErrOtpCheckRestrict
	ErrOtpCheckFail
	ErrAccountNotAvailable
	LoginMsg
	ErrUsernameOrPwdErr
	ErrOauth2NoAutoRegister
)
