package base

import (
	"encoding/json"
	"fmt"
)

const (
	SuccessCode = 200
	SuccessMsg  = "success"

	BizErrorCode = 400
	BizErrorMsg  = "error"

	ServerErrorCode = 500
	ServerErrorMsg  = "server error"

	TokenErrorCode = 501
	TokenErrorMsg  = "token error"
)

// 统一返回结果结构体
type Result struct {
	Code int16       `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// 将Result转为json字符串
func (r *Result) ToJson() string {
	jsonData, err := json.Marshal(r)
	if err != nil {
		fmt.Println("data转json错误")
	}
	return string(jsonData)
}

// 判断该Result是否为成功状态
func (r *Result) IsSuccess() bool {
	return r.Code == SuccessCode
}

// 返回成功状态的Result
// @param data 成功附带的数据消息
func Success(data interface{}) *Result {
	return &Result{Code: SuccessCode, Msg: SuccessMsg, Data: data}
}

// 返回成功状态的Result
// @param data 成功不附带数据
func SuccessNoData() *Result {
	return &Result{Code: SuccessCode, Msg: SuccessMsg}
}

// 返回服务器错误Result
func ServerError() *Result {
	return &Result{Code: ServerErrorCode, Msg: ServerErrorMsg}
}

func Error(code int16, msg string) *Result {
	return &Result{Code: code, Msg: msg}
}

func TokenError() *Result {
	return &Result{Code: TokenErrorCode, Msg: TokenErrorMsg}
}
