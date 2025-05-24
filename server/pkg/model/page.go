package model

import (
	"mayfly-go/pkg/utils/structx"
)

// 分页参数
type PageParam struct {
	PageNum  int `json:"pageNum" form:"pageNum" gorm:"-"`
	PageSize int `json:"pageSize" form:"pageSize" gorm:"-"`
}

// 分页结果
type PageResult[T any] struct {
	Total int64 `json:"total"`
	List  []T   `json:"list"`
}

// 空分页结果
func NewEmptyPageResult[T any]() *PageResult[T] {
	return &PageResult[T]{Total: 0}
}

// PageResultConv pageResult转换
func PageResultConv[F any, T any](pageResult *PageResult[F]) *PageResult[T] {
	if pageResult == nil {
		return NewEmptyPageResult[T]()
	}
	return &PageResult[T]{
		Total: pageResult.Total,
		List:  structx.CopySliceTo[F, T](pageResult.List),
	}
}
