package model

// 分页参数
type PageParam struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

// 分页结果
type PageResult[T any] struct {
	Total int64 `json:"total"`
	List  T     `json:"list"`
}

// 空分页结果
func EmptyPageResult[T any]() *PageResult[T] {
	return &PageResult[T]{Total: 0}
}
