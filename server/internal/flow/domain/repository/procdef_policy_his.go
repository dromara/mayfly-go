package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
)

// ProcdefPolicyHis 触发策略变更记录仓储
type ProcdefPolicyHis interface {
	base.Repo[*entity.ProcdefPolicyHis]

	// ListByProcdef 按时间倒序返回某流程定义的策略变更记录
	ListByProcdef(procdefId uint64, limit int) ([]*entity.ProcdefPolicyHis, error)
}
