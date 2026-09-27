//go:build it

package itest

// 保存期增量字段索引校验（P1）的真库行为：
//   - 主键列兜底：pg/mssql/oracle/dm/sqlite 方言的 GetTableIndex 有意排除主键索引（仅 mysql 含 PRIMARY 行），
//     UpdField 为主键时必须经 GetPrimaryKeys 兜底放行，否则仅含主键的表在 pg 源上会被误拒；
//   - 无索引且非主键首列的列仍须拒绝，兜底不得把真无索引的列放进来。
//
// 本地仅提供 mysql/pg 容器，mssql/oracle/dm/sqlite 的「排除主键」口径由各自 meta.sql 保证，不在本用例覆盖内。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/domain/entity"
)

func pkOnlyTask(srcTable, updField string) *entity.DataSyncTask {
	return &entity.DataSyncTask{
		SyncMode:        entity.DataSyncModeIncrementalMerge,
		TargetTableName: "it_sync_pk_tgt",
		UpdField:        updField,
		DataSQL:         "SELECT id, name FROM " + srcTable,
	}
}

// TestITIndexValidationPrimaryKeyOnly 仅含主键（无二级索引）的源表，UpdField 配主键列应放行
func TestITIndexValidationPrimaryKeyOnly(t *testing.T) {
	ctx := context.Background()
	table := "it_sync_pk_only"

	// pg：GetTableIndex 排除 _pkey 主键索引，是主键兜底的核心回归场景（修复前此处误拒）
	pg := pgConn(t)
	t.Cleanup(func() { pg.Close() })
	mustExec(t, pg, "DROP TABLE IF EXISTS "+quote(pg, table))
	mustExec(t, pg, "CREATE TABLE "+quote(pg, table)+" (id bigint PRIMARY KEY, name varchar(50))")
	require.NoError(t, sync.ValidateIncrementalFieldIndex(ctx, pg, pkOnlyTask(table, "id")),
		"pg 源仅有主键时 UpdField=id 不应被拒绝")

	// mysql：GetTableIndex 含 PRIMARY 行，主键列天然命中（回归保护，防止兜底改动破坏原有路径）
	my := mysqlConn(t)
	t.Cleanup(func() { my.Close() })
	mustExec(t, my, "DROP TABLE IF EXISTS "+quote(my, table))
	mustExec(t, my, "CREATE TABLE "+quote(my, table)+" (id bigint PRIMARY KEY, name varchar(50))")
	require.NoError(t, sync.ValidateIncrementalFieldIndex(ctx, my, pkOnlyTask(table, "id")))

	// 非主键且无索引的列仍须拒绝：兜底只放行主键首列
	err := sync.ValidateIncrementalFieldIndex(ctx, my, pkOnlyTask(table, "name"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not the leading column")
}
