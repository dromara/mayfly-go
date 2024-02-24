package req

import (
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
	"mayfly-go/pkg/validatorx"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// 绑定并校验请求结构体参数
func BindJsonAndValid[T any](rc *Ctx, data T) T {
	if err := rc.F.BindJSON(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data
	}
}

// 绑定请求体中的json至form结构体，并拷贝至另一结构体
func BindJsonAndCopyTo[T any](rc *Ctx, form any, toStruct T) T {
	BindJsonAndValid(rc, form)
	structx.Copy(toStruct, form)
	return toStruct
}

// 绑定查询字符串到指定结构体
func BindQuery[T any](rc *Ctx, data T) T {
	if err := rc.F.BindQuery(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data
	}
}

// 绑定查询字符串到指定结构体，并将分页信息也返回
func BindQueryAndPage[T any](rc *Ctx, data T) (T, *model.PageParam) {
	if err := rc.F.BindQuery(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data, rc.F.GetPageParam()
	}
}

// 返回失败结果集
func ErrorRes(rc *Ctx, err any) {
	switch t := err.(type) {
	case errorx.BizError:
		rc.F.JSONRes(http.StatusOK, model.Error(t))
	default:
		logx.ErrorTrace("服务器错误", t)
		rc.F.JSONRes(http.StatusOK, model.ServerError())
	}
}

// 转换参数校验错误为业务异常错误
func ConvBindValidationError(data any, err error) error {
	if e, ok := err.(validator.ValidationErrors); ok {
		return errorx.NewBizCode(403, validatorx.Translate2Str(data, e))
	}
	return err
}
