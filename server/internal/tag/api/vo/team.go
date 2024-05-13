package vo

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"time"
)

type Team struct {
	model.Model
	entity.RelateTags // 标签信息

	Name              string          `json:"name"`              // 名称
	ValidityStartDate *model.JsonTime `json:"validityStartDate"` // 生效开始时间
	ValidityEndDate   *model.JsonTime `json:"validityEndDate"`   // 生效结束时间
	Remark            string          `json:"remark"`            // 备注说明
}

func (t *Team) GetRelateId() uint64 {
	return t.Id
}

// 团队成员信息
type TeamMember struct {
	Id         uint64     `json:"id"`
	TeamId     uint64     `json:"teamId"`
	AccountId  uint64     `json:"accountId"`
	Username   string     `json:"username"`
	Name       string     `json:"name"`
	Creator    string     `json:"creator"`
	CreateTime *time.Time `json:"createTime"`
}
