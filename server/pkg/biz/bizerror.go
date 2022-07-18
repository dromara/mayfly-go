package biz

// 业务错误
type BizError struct {
	code int16
	err  string
}

var (
	Success       BizError = NewBizErrCode(200, "success")
	BizErr        BizError = NewBizErrCode(400, "biz error")
	ServerError   BizError = NewBizErrCode(500, "server error")
	PermissionErr BizError = NewBizErrCode(501, "token error")
)

// 错误消息
func (e BizError) Error() string {
	return e.err
}

// 错误码
func (e BizError) Code() int16 {
	return e.code
}

// 创建业务逻辑错误结构体，默认为业务逻辑错误
func NewBizErr(msg string) BizError {
	return BizError{code: BizErr.code, err: msg}
}

// 创建业务逻辑错误结构体，可设置指定错误code
func NewBizErrCode(code int16, msg string) BizError {
	return BizError{code: code, err: msg}
}
