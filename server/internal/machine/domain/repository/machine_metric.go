package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
)

type MachineMetric interface {
	base.Repo[*entity.MachineMetric]

	// LatestPerMachine 每台机器最新一条指标（健康总览用，单条聚合查询避免 N+1）
	LatestPerMachine() ([]*entity.MachineMetric, error)
}
