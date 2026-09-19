package transfer

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
)

// Db2DbTransferStrategy DB→DB 迁移策略：将源库数据迁移到目标库。
type Db2DbTransferStrategy struct {
	app *DbTransferAppImpl
}

// NewDb2DbTransferStrategy 创建 DB→DB 迁移策略
func NewDb2DbTransferStrategy(app *DbTransferAppImpl) *Db2DbTransferStrategy {
	return &Db2DbTransferStrategy{app: app}
}

// Name 返回策略名称
func (s *Db2DbTransferStrategy) Name() string {
	return "db2db"
}

// Execute 执行 DB→DB 迁移
func (s *Db2DbTransferStrategy) Execute(ctx context.Context, pctx *PipelineContext) error {
	// 委托给现有的 transfer2Db 方法（保持向后兼容）
	s.app.transfer2Db(ctx, pctx.LogId, pctx.Task, pctx.SrcConn, pctx.Tables)
	return nil
}

// Db2FileTransferStrategy DB→File 迁移策略：将源库数据导出为文件。
type Db2FileTransferStrategy struct {
	app *DbTransferAppImpl
}

// NewDb2FileTransferStrategy 创建 DB→File 迁移策略
func NewDb2FileTransferStrategy(app *DbTransferAppImpl) *Db2FileTransferStrategy {
	return &Db2FileTransferStrategy{app: app}
}

// Name 返回策略名称
func (s *Db2FileTransferStrategy) Name() string {
	return "db2file"
}

// Execute 执行 DB→File 迁移
func (s *Db2FileTransferStrategy) Execute(ctx context.Context, pctx *PipelineContext) error {
	// 委托给现有的 transfer2File 方法（保持向后兼容）
	s.app.transfer2File(ctx, pctx.LogId, pctx.Task, pctx.Tables)
	return nil
}

// DefaultTransferStrategyRegistry 创建默认的策略注册表，注册所有内置迁移模式。
func DefaultTransferStrategyRegistry(app *DbTransferAppImpl) *TransferStrategyRegistry {
	registry := NewTransferStrategyRegistry()
	registry.Register(entity.DbTransferTaskModeDb, NewDb2DbTransferStrategy(app))
	registry.Register(entity.DbTransferTaskModeFile, NewDb2FileTransferStrategy(app))
	return registry
}
