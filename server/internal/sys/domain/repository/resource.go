package repository

import (
	"context"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
)

type Resource interface {
	base.Repo[*entity.Resource]

	// 获取账号资源列表
	GetAccountResources(ctx context.Context, accountId uint64, toEntity any) error

	// 获取所有子节点id
	GetChildren(ctx context.Context, uiPath string) ([]entity.Resource, error)

	// 根据uiPath右匹配更新所有相关类资源
	UpdateByUiPathLike(ctx context.Context, resource *entity.Resource) error
}
