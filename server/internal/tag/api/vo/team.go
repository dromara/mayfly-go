package vo

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"time"
)

type Team struct {
	model.Model
	entity.RelateTags // 标签信息

	Name              string     `json:"name"`              // 名称
	ValidityStartDate *time.Time `json:"validityStartDate"` // 生效开始时间
	ValidityEndDate   *time.Time `json:"validityEndDate"`   // 生效结束时间
	Remark            string     `json:"remark"`            // 备注说明
}

func (t *Team) GetRelateId() uint64 {
	return t.Id
}
