package transfer

import (
	"context"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

// TransferStrategy 迁移策略接口：定义不同迁移模式（DB→DB / DB→File）的核心行为。
// 遵循开闭原则：新增迁移模式只需实现此接口并注册到策略注册表，无需修改调度逻辑。
type TransferStrategy interface {
	// Name 返回策略名称（用于日志）
	Name() string

	// Execute 执行迁移策略
	Execute(ctx context.Context, pctx *PipelineContext) error
}

// PipelineContext 迁移流水线上下文：封装迁移执行期所需的全部共享状态。
// 各阶段通过此结构传递数据，避免参数列表过长。
type PipelineContext struct {
	// Task 迁移任务实体
	Task *entity.DbTransferTask
	// TaskId 任务ID（快捷引用）
	TaskId uint64
	// LogId 执行日志ID
	LogId uint64
	// SrcConn 源库连接（可能为nil，如file→db模式）
	SrcConn *dbi.DbConn
	// Tables 待迁移的表列表
	Tables []dbi.Table
}

// TransferStrategyRegistry 迁移策略注册表：按模式索引的策略集合。
// 遵循开闭原则：新增模式只需 Register，无需修改调度逻辑。
type TransferStrategyRegistry struct {
	strategies map[int8]TransferStrategy
}

// NewTransferStrategyRegistry 创建策略注册表
func NewTransferStrategyRegistry() *TransferStrategyRegistry {
	return &TransferStrategyRegistry{
		strategies: make(map[int8]TransferStrategy),
	}
}

// Register 注册迁移策略
func (r *TransferStrategyRegistry) Register(mode int8, strategy TransferStrategy) {
	r.strategies[mode] = strategy
}

// Get 获取指定模式的迁移策略
func (r *TransferStrategyRegistry) Get(mode int8) (TransferStrategy, bool) {
	s, ok := r.strategies[mode]
	return s, ok
}

// TransferPipeline 迁移流水线：编排迁移执行的各阶段。
// 职责：连接管理 → 表信息获取 → 策略执行 → 进度汇总。
type TransferPipeline struct {
	registry *TransferStrategyRegistry
}

// NewTransferPipeline 创建迁移流水线
func NewTransferPipeline(registry *TransferStrategyRegistry) *TransferPipeline {
	return &TransferPipeline{registry: registry}
}

// Execute 执行迁移流水线
func (p *TransferPipeline) Execute(ctx context.Context, pctx *PipelineContext) error {
	// 查找策略
	strategy, ok := p.registry.Get(pctx.Task.Mode)
	if !ok {
		return &UnsupportedModeError{Mode: pctx.Task.Mode}
	}

	// 执行策略
	return strategy.Execute(ctx, pctx)
}

// UnsupportedModeError 不支持的迁移模式错误
type UnsupportedModeError struct {
	Mode int8
}

func (e *UnsupportedModeError) Error() string {
	return "unsupported transfer mode"
}
