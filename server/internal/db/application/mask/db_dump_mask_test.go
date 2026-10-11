package mask

// 导出侧行级脱敏钩子的回归测试。
//
// 覆盖三条不变量：
//  1. 按表只构建一次脱敏器（导出是逐行回调，每行重建会把规则解析开销摊到每一行）；
//  2. 「该表无需脱敏」也要缓存（nil 是有效结论，不是未计算）；
//  3. 脱敏计划构建失败时不缓存失败结果、也不降级为不脱敏——错误必须一路上抛中止导出。
//
// 引擎用桩替代：判定口径（启用开关/豁免角色/fail-close）由 MaskEngine 自身负责并另有真库集成测试，
// 本文件只钉住钩子与缓存这条编排逻辑

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	masksvc "mayfly-go/internal/db/domain/mask"
)

// stubMaskEngine 只实现 MaskEngine 接口，按表返回预设结论并记录构建次数
type stubMaskEngine struct {
	builds map[string]int
	// resultByTable 表名 -> 预置返回（masker 与 err 二选一）
	maskerByTable map[string]*masksvc.RowMasker
	errByTable    map[string]error
}

func newStubEngine() *stubMaskEngine {
	return &stubMaskEngine{
		builds:        make(map[string]int),
		maskerByTable: make(map[string]*masksvc.RowMasker),
		errByTable:    make(map[string]error),
	}
}

var _ MaskEngine = (*stubMaskEngine)(nil)

func (s *stubMaskEngine) MaskValue(_ context.Context, _ uint64, _, _, _ string, value any) (any, error) {
	return value, nil
}

func (s *stubMaskEngine) MaskRowMap(context.Context, uint64, string, string, map[string]any) error {
	return nil
}

func (s *stubMaskEngine) BuildStmtRowMasker(context.Context, *dbi.DbConn, sqlstmt.Stmt, []*dbi.QueryColumn) (*masksvc.RowMasker, error) {
	return nil, nil
}

func (s *stubMaskEngine) BuildQueryRowMasker(context.Context, *dbi.DbConn, string, []*dbi.QueryColumn) (*masksvc.RowMasker, error) {
	return nil, nil
}

func (s *stubMaskEngine) BuildTableColumnMasker(_ context.Context, _ *dbi.DbConn, _, table string, _ []string) (*masksvc.RowMasker, error) {
	s.builds[table]++
	if err, ok := s.errByTable[table]; ok {
		return nil, err
	}
	return s.maskerByTable[table], nil
}

// phoneMasker 构造一个真实的行脱敏器（full 算法整列置掩码），避免用假对象掩盖 MaskRow 的行为
func phoneMasker(t *testing.T) *masksvc.RowMasker {
	t.Helper()
	alg, err := masksvc.Get(masksvc.AlgoFull)
	require.NoError(t, err)
	plan, err := masksvc.NewPlan([]*masksvc.Rule{{
		Name:      "手机号",
		MatchType: masksvc.MatchTypeExact,
		Pattern:   "phone",
		Algorithm: alg,
	}}, nil)
	require.NoError(t, err)
	return masksvc.NewRowMasker(plan, "testdb", []string{"t_user"}, nil, []string{"phone", "name"}, nil, nil)
}

func TestDumpRowMaskerCachesPerTable(t *testing.T) {
	engine := newStubEngine()
	engine.maskerByTable["t_user"] = phoneMasker(t)

	hook := NewDumpRowMasker(engine, nil, "testdb").Fn()
	require.NotNil(t, hook)

	// 同一张表连续三行：只应构建一次脱敏器
	for i := range 3 {
		row := map[string]any{"id": i, "phone": "13800001234", "name": "张三"}
		require.NoError(t, hook(context.Background(), "t_user", []string{"id", "phone", "name"}, row))
		// 每行都必须真的被脱敏（缓存的是脱敏器本身，不是首行结果）
		assert.NotEqual(t, "13800001234", row["phone"], "phone 列必须被脱敏")
		assert.Equal(t, "张三", row["name"], "未命中规则的列不得被改动")
	}
	assert.Equal(t, 1, engine.builds["t_user"], "同表多次回调只构建一次脱敏器")
}

// 无规则命中（masker 为 nil）同样是结论，必须缓存：否则每次导出都对无敏感列的表重复解析一遍
func TestDumpRowMaskerCachesNoMaskConclusion(t *testing.T) {
	engine := newStubEngine() // t_log 未预置任何结论 → 返回 (nil, nil)
	hook := NewDumpRowMasker(engine, nil, "testdb").Fn()
	require.NotNil(t, hook)

	for i := range 2 {
		row := map[string]any{"id": i, "msg": "hello"}
		require.NoError(t, hook(context.Background(), "t_log", []string{"id", "msg"}, row))
		assert.Equal(t, "hello", row["msg"], "无脱敏器时行内容保持原样")
	}
	assert.Equal(t, 1, engine.builds["t_log"], "「无需脱敏」也必须只判定一次")
}

// fail-close：构建失败必须上抛，且不得把失败当结论缓存（否则一次瞬时错误会让整场导出都不脱敏）
func TestDumpRowMaskerFailCloseNotCached(t *testing.T) {
	engine := newStubEngine()
	engine.errByTable["t_user"] = errors.New("mask plan unavailable")

	hook := NewDumpRowMasker(engine, nil, "testdb").Fn()
	require.NotNil(t, hook)

	row := map[string]any{"phone": "13800001234"}
	assert.Error(t, hook(context.Background(), "t_user", []string{"phone"}, row))
	assert.Error(t, hook(context.Background(), "t_user", []string{"phone"}, row))
	assert.Equal(t, 2, engine.builds["t_user"], "失败结果不得缓存，下次仍应重新判定")
}

// 引擎缺失时返回 nil 钩子（导出器据此跳过钩子），而不是返回一个「总是报错」的钩子：
// 脱敏是可选能力，能力不存在不等于导出失败
func TestDumpRowMaskerNilEngine(t *testing.T) {
	assert.Nil(t, NewDumpRowMasker(nil, nil, "testdb").Fn())
}
