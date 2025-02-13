package entity

import (
	"mayfly-go/pkg/model"
)

// 团队信息
type Team struct {
	model.Model

	Name              string          `json:"name" gorm:"not null;size:36;comment:名称"` // 名称
	ValidityStartDate *model.JsonTime `json:"validityStartDate" gorm:"comment:生效开始时间"` // 生效开始时间
	ValidityEndDate   *model.JsonTime `json:"validityEndDate" gorm:"comment:生效结束时间"`   // 生效结束时间
	Remark            string          `json:"remark" gorm:"size:255;"`                 // 备注说明
}
