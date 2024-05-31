package entity

import (
	"mayfly-go/pkg/model"
)

// 团队信息
type Team struct {
	model.Model

	Name              string          `json:"name"`              // 名称
	ValidityStartDate *model.JsonTime `json:"validityStartDate"` // 生效开始时间
	ValidityEndDate   *model.JsonTime `json:"validityEndDate"`   // 生效结束时间
	Remark            string          `json:"remark"`            // 备注说明
}
