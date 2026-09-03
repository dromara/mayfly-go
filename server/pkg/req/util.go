package req

import (
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
	"mayfly-go/pkg/validatorx"

	"github.com/go-playground/validator/v10"
)

// BindJson 绑定并校验请求结构体参数
func (rc *Ctx) BindJson[T any]() *T {
	var data T
	if err := rc.BindJSON(&data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return &data
	}
}

// BindJsonAndCopyTo 绑定请求体中的json至form结构体，并拷贝至指定结构体
func (rc *Ctx) BindJsonAndCopyTo[F, T any]() (*F, *T) {
	f := rc.BindJson[F]()
	return f, structx.CopyTo[T](f)
}

// BindQuery 绑定查询字符串到指定结构体
func (rc *Ctx) BindQuery[T any]() *T {
	var data T
	// Ctx.BindQuery为泛型方法，此处需显式调用wrapperF的非泛型BindQuery
	if err := rc.wrapperF.BindQuery(&data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return &data
	}
}

// BindQueryAndPage 绑定查询字符串到指定结构体，并将分页信息也返回
func (rc *Ctx) BindQueryAndPage[T any]() (*T, model.PageParam) {
	var data T
	if err := rc.wrapperF.BindQuery(&data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return &data, rc.GetPageParam()
	}
}

// 转换参数校验错误为业务异常错误
func ConvBindValidationError(data any, err error) error {
	if e, ok := err.(validator.ValidationErrors); ok {
		return errorx.NewBizCode(403, validatorx.Translate2Str(data, e))
	}
	return err
}
