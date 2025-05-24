package req

import (
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
	"mayfly-go/pkg/validatorx"

	"github.com/go-playground/validator/v10"
)

// 绑定并校验请求结构体参数
func BindJsonAndValid[T any](rc *Ctx) T {
	data := structx.NewInstance[T]()
	if err := rc.BindJSON(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data
	}
}

// 绑定请求体中的json至form结构体，并拷贝至指定结构体
func BindJsonAndCopyTo[F, T any](rc *Ctx) (F, T) {
	f := BindJsonAndValid[F](rc)
	return f, structx.CopyTo[T](f)
}

// 绑定查询字符串到指定结构体
func BindQuery[T any](rc *Ctx) T {
	data := structx.NewInstance[T]()
	if err := rc.BindQuery(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data
	}
}

// 绑定查询字符串到指定结构体，并将分页信息也返回
func BindQueryAndPage[T any](rc *Ctx) (T, model.PageParam) {
	data := structx.NewInstance[T]()
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
