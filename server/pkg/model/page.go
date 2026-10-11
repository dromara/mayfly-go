package model

import (
	"mayfly-go/pkg/utils/structx"
)

const (
	// DefaultPageSize 未指定页大小时每页取回的条数
	DefaultPageSize = 100
	// MaxPageSize 单页允许取回的最大条数：分页入口暴露的是 HTTP 参数，不设上限会让一次请求
	// 拉走整表，把内存与连接一起打满
	MaxPageSize = 2000
	// DefaultPageNum 未指定页码时的页码
	DefaultPageNum = 1
)

// 分页参数
type PageParam struct {
	PageNum  int `json:"pageNum" form:"pageNum" gorm:"-"`
	PageSize int `json:"pageSize" form:"pageSize" gorm:"-"`
}

// Normalize 返回实际生效的页码与页大小（零值补默认、超上限夹紧）
func (p PageParam) Normalize() (pageNum, pageSize int) {
	pageNum, pageSize = p.PageNum, p.PageSize
	if pageNum <= 0 {
		pageNum = DefaultPageNum
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return
}

// 分页结果
type PageResult[T any] struct {
	Total int64 `json:"total"`
	List  []T   `json:"list"`

	// PageNum/PageSize 为本页实际生效值。请求超出上限时被夹小的部分必须回传，
	// 否则调用方与前端会把「被截断」当成「数据就只有这么多」
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

// 空分页结果。
//
// List 给空切片而不是 nil：它会被直接序列化回前端，nil 会变成 null，于是每个消费点都要各自
// 防一遍 null——与 PageQuery 的空结果保持同一口径
func NewEmptyPageResult[T any]() *PageResult[T] {
	return &PageResult[T]{Total: 0, List: []T{}, PageNum: DefaultPageNum, PageSize: DefaultPageSize}
}

// NewEmptyPageResultOf 按请求的分页参数给一个空结果。
//
// 用于「条件就不可能命中」的早退分支：页码与页大小回报实际生效值，前端才能继续正常渲染分页条
func NewEmptyPageResultOf[T any](pageParam PageParam) *PageResult[T] {
	pageNum, pageSize := pageParam.Normalize()
	return &PageResult[T]{Total: 0, List: []T{}, PageNum: pageNum, PageSize: pageSize}
}

// PageResultConv pageResult转换
func PageResultConv[F any, T any](pageResult *PageResult[F]) *PageResult[T] {
	if pageResult == nil {
		return NewEmptyPageResult[T]()
	}
	list := structx.CopySliceTo[F, T](pageResult.List)
	if list == nil {
		list = []T{}
	}
	return &PageResult[T]{
		Total:    pageResult.Total,
		List:     list,
		PageNum:  pageResult.PageNum,
		PageSize: pageResult.PageSize,
	}
}
