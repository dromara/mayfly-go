package model

// 分页参数
type PageParam struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

// 分页结果
type PageResult struct {
	Total int64 `json:"total"`
	List  any   `json:"list"`
}

// 空分页结果日志
func EmptyPageResult() *PageResult {
	return &PageResult{Total: 0, List: make([]any, 0)}
}
