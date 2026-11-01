package sqlexec

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeDumpTableAcl 只实现导出判定用到的方法
type fakeDumpTableAcl struct {
	TableAcl
	gotDbId       uint64
	gotTbls       []string
	tableLevelHit bool
	tableErr      error
	err           error
}

// AssertTablesAllowed 表级闸门（deny 名单 / 白名单）
func (f *fakeDumpTableAcl) AssertTablesAllowed(ctx context.Context, dbId uint64, tables []string) error {
	f.tableLevelHit = true
	return f.tableErr
}

func (f *fakeDumpTableAcl) AssertWholeTableAllowed(ctx context.Context, dbId uint64, tables []string) error {
	f.gotDbId = dbId
	f.gotTbls = tables
	return f.err
}

// TestAssertDumpTablesAllowedIsTheSingleGate 导出权限判定单点：命中即拒绝，未配置即放行。
//
// 历史缺陷：判定写在 api 层的 `if needData` 内，DumpType=1（只导结构）整段跳过 ——
// 受保护表的表名/列名/列类型/默认值/注释可被完整取走。抽成单点后，
// "任何导出形态都先过这里"由调用位置守卫 + 本行为测试共同锁定。
func TestAssertDumpTablesAllowedIsTheSingleGate(t *testing.T) {
	denied := errors.New("table protected by ACL")
	acl := &fakeDumpTableAcl{err: denied}

	err := AssertDumpTablesAllowed(context.Background(), acl, 42, []string{"t_secret"})
	require.ErrorIs(t, err, denied, "命中表/列保护必须原样返回拒绝原因")
	require.EqualValues(t, 42, acl.gotDbId)
	require.Equal(t, []string{"t_secret"}, acl.gotTbls, "表清单必须原样交给判定（含仅结构导出的清单）")
	require.True(t, acl.tableLevelHit,
		"表级闸门必须单独判：AssertWholeTableAllowed 只查列规则，只调它会让表级保护在导出通道失效")

	// 只配表级保护（无列规则）也必须拒绝
	tableOnly := &fakeDumpTableAcl{tableErr: errors.New("table denied")}
	err = AssertDumpTablesAllowed(context.Background(), tableOnly, 42, []string{"t_secret"})
	require.Error(t, err, "仅表级 deny 时导出必须被拒绝")
	require.Contains(t, err.Error(), "table denied")

	// 未配置权限（acl 为 nil）不应拦
	require.NoError(t, AssertDumpTablesAllowed(context.Background(), nil, 42, []string{"t"}))
}
