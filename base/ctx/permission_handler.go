package ctx

import (
	"mayfly-go/base/biz"
)

type Permission struct {
	Code        string // 权限code
	Description string // 请求描述
}

var permissionError = biz.NewBizErrCode(biz.TokenErrorCode, biz.TokenErrorMsg)

func PermissionHandler(rc *ReqCtx) error {
	if !rc.NeedToken {
		return nil
	}
	tokenStr := rc.GinCtx.Request.Header.Get("Authorization")
	if tokenStr == "" {
		tokenStr = rc.GinCtx.Query("token")
	}
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
