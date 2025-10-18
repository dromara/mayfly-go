package errorx

import (
	"context"
	"fmt"
	"mayfly-go/pkg/i18n"
)

// 业务错误
type BizError struct {
	code int16
	err  string
}

var (
	Success            *BizError = NewBizCode(200, "success")
	BizErr             *BizError = NewBizCode(400, "biz error")
	ServerError        *BizError = NewBizCode(500, "server error")
	PermissionErr      *BizError = NewBizCode(501, "token error")
	AccessTokenInvalid *BizError = NewBizCode(502, "access token invalid")
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

// NewBiz 创建业务逻辑错误结构体，默认为业务逻辑错误
func NewBiz(msg string) *BizError {
	return &BizError{code: BizErr.code, err: msg}
}

// NewBizf 创建业务逻辑错误结构体，可设置格式化参数
func NewBizf(format string, formatValues ...any) *BizError {
	return NewBiz(fmt.Sprintf(format, formatValues...))
}

// NewBizI 使用i18n的msgId创建业务逻辑错误结构体，默认为业务逻辑错误 (使用ctx中的国际化语言)
//
//	// NameErr =  {{.name}} is invalid  =>  xxx is invalid
//	NewBizI(ctx, imsg.NameErr, "name", "xxxx")
func NewBizI(ctx context.Context, msgId i18n.MsgId, attrs ...any) *BizError {
	return &BizError{code: BizErr.code, err: i18n.TC(ctx, msgId, attrs...)}
}

// NewBizCode 创建业务逻辑错误结构体，可设置指定错误code
func NewBizCode(code int16, msg string) *BizError {
	return &BizError{code: code, err: msg}
}

// NewBizCodef 创建业务逻辑错误结构体，可设置指定错误code，并且支持格式化参数
func NewBizCodef(code int16, format string, formats ...any) *BizError {
	return NewBizCode(code, fmt.Sprintf(format, formats...))
}
