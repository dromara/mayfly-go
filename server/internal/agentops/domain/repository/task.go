package repository

import (
	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

// AgentTask 定时计划任务仓储
type AgentTask interface {
	base.Repo[*entity.AgentTask]

	// GetPageList 分页查询任务（按名称模糊、状态、装配档、归属账号过滤）
	GetPageList(condition *entity.AgentTask, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.AgentTask], error)
}
