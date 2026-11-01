package transfer

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/internal/db/domain/entity"

	"github.com/stretchr/testify/require"
)

// fakeStrategy 记录是否被调用，并按需返回错误
type fakeStrategy struct {
	called bool
	err    error
}

func (s *fakeStrategy) Name() string { return "fake" }
func (s *fakeStrategy) Execute(ctx context.Context, pctx *PipelineContext) error {
	s.called = true
	return s.err
}

// TestPipelineUnsupportedModeReturnsTypedError 未注册模式必须返回 *UnsupportedModeError。
//
// Run 依赖这个类型区分「模式未注册」与「策略自身失败」：前者按 unsupported 收尾，
// 后者绝不能复用该文案（否则真实原因被覆盖成"不支持的模式"）。
func TestPipelineUnsupportedModeReturnsTypedError(t *testing.T) {
	p := &TransferPipeline{registry: NewTransferStrategyRegistry()}
	err := p.Execute(context.Background(), &PipelineContext{Task: &entity.DbTransferTask{Mode: entity.TransferMode(99)}})

	var unsupported *UnsupportedModeError
	require.True(t, errors.As(err, &unsupported), "必须返回可识别的 UnsupportedModeError，实际: %v", err)
	require.Equal(t, entity.TransferMode(99), unsupported.Mode)
}

// TestPipelineDelegatesToRegisteredStrategy 已注册模式必须原样把错误透传给调用方
// （由 Run 决定收尾方式，Pipeline 不得自行吞掉或改写）。
func TestPipelineDelegatesToRegisteredStrategy(t *testing.T) {
	registry := NewTransferStrategyRegistry()
	boom := errors.New("table not found")
	strategy := &fakeStrategy{err: boom}
	registry.Register(entity.DbTransferTaskModeDb, strategy)

	p := &TransferPipeline{registry: registry}
	err := p.Execute(context.Background(), &PipelineContext{Task: &entity.DbTransferTask{Mode: entity.DbTransferTaskModeDb}})

	require.True(t, strategy.called, "已注册策略必须被调用")
	require.ErrorIs(t, err, boom, "策略错误必须原样透传，不能被包装成 unsupported")
}
