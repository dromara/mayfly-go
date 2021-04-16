package ctx

import (
	"mayfly-go/base/biz"
)

func init() {
	BeforeHandlers = append(BeforeHandlers, new(PermissionHandler))
}

var permissionError = biz.NewBizErrCode(501, "token error")

type PermissionHandler struct{}

func (p *PermissionHandler) BeforeHandle(rc *ReqCtx) error {
	if !rc.NeedToken {
		return nil
	}
	tokenStr := rc.Req.Header.Get("Authorization")
	if tokenStr == "" {
		return permissionError
	}
	loginAccount, err := ParseToken(tokenStr)
	if err != nil || loginAccount == nil {
		return permissionError
	}
	rc.LoginAccount = loginAccount
	return nil
}
