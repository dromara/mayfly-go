package errorx

import (
	"fmt"
)

// 业务错误
type BizError struct {
	code int16
	err  string
}

var (
	Success       BizError = NewBizCode(200, "success")
	BizErr        BizError = NewBizCode(400, "biz error")
	ServerError   BizError = NewBizCode(500, "server error")
	PermissionErr BizError = NewBizCode(501, "token error")
)

// 错误消息
func (e BizError) Error() string {
	return e.err
}

// 错误码
func (e BizError) Code() int16 {
	return e.code
}

func (e BizError) String() string {
	return fmt.Sprintf("errCode: %d, errMsg: %s", e.Code(), e.Error())
}

// 创建业务逻辑错误结构体，默认为业务逻辑错误
func NewBiz(msg string, formats ...any) BizError {
	return BizError{code: BizErr.code, err: fmt.Sprintf(msg, formats...)}
}

// 创建业务逻辑错误结构体，可设置指定错误code
func NewBizCode(code int16, msg string, formats ...any) BizError {
	return BizError{code: code, err: fmt.Sprintf(msg, formats...)}
}
