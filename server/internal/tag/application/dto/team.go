package dto

import "mayfly-go/pkg/model"

type SaveTeam struct {
	Id                uint64          `json:"id"`
	Name              string          `json:"name" binding:"required"` // 名称
	ValidityStartDate *model.JsonTime `json:"validityStartDate"`       // 生效开始时间
	ValidityEndDate   *model.JsonTime `json:"validityEndDate"`         // 生效结束时间
	Remark            string          `json:"remark"`                  // 备注说明

	CodePaths []string `json:"codePaths"` // 关联标签信息
}
