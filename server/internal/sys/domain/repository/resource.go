package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
)

type Resource interface {
	base.Repo[*entity.Resource]

	// 获取账号资源列表
	GetAccountResources(accountId uint64, toEntity any) error

	// 获取所有子节点id
	GetChildren(uiPath string) []entity.Resource

	// 根据uiPath右匹配更新所有相关类资源
	UpdateByUiPathLike(resource *entity.Resource) error
}
