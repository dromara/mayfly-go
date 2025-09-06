package repository

import (
	"mayfly-go/internal/docker/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Container interface {
	base.Repo[*entity.Container]

	// 分页获取容器配置列表
	GetContainerPage(condition *entity.ContainerQuery, orderBy ...string) (*model.PageResult[*entity.Container], error)
}
