package dto

type SaveTeam struct {
	Id     uint64 `json:"id"`
	Name   string `json:"name" binding:"required"` // 名称
	Remark string `json:"remark"`                  // 备注说明

	CodePaths []string `json:"codePaths"` // 关联标签信息
}
