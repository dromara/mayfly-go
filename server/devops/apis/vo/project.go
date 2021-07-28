package vo

// 用户选择项目
type AccountProject struct {
	Id     uint64 `json:"id"`
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

type AccountProjects []AccountProject
