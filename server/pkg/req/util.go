package req

import (
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
	"mayfly-go/pkg/validatorx"

	"github.com/go-playground/validator/v10"
)

// 绑定并校验请求结构体参数
func BindJsonAndValid[T any](rc *Ctx, data T) T {
	if err := rc.BindJSON(data); err != nil {
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
	if err := rc.BindQuery(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data
	}
}

// 绑定查询字符串到指定结构体，并将分页信息也返回
func BindQueryAndPage[T any](rc *Ctx, data T) (T, *model.PageParam) {
	if err := rc.BindQuery(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data, rc.GetPageParam()
	}
}

// 转换参数校验错误为业务异常错误
func ConvBindValidationError(data any, err error) error {
	if e, ok := err.(validator.ValidationErrors); ok {
		return errorx.NewBizCode(403, validatorx.Translate2Str(data, e))
	}
	return err
}
