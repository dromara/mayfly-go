package repository

import (
	"context"

	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type TeamMember interface {
	base.Repo[*entity.TeamMember]

	// 获取项目成员列表
	ListMemeber(condition *entity.TeamMember, toEntity []any, orderBy ...string)

	GetPageList(condition *entity.TeamMember, pageParam model.PageParam) (*model.PageResult[*entity.TeamMemberPO], error)

	// 是否存在指定的团队成员关联信息
	//
	// 必须回报错误：统计失败被当成「不存在」会放行重复成员，而成员唯一性没有数据库索引兜底
	IsExist(ctx context.Context, teamId, accountId uint64) (bool, error)
}
