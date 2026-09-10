package itest

// sqlite「约束生成的隐式索引」dump 保真集成测试。
//
// 背景（本机实测）：sqlite 中 PRIMARY KEY / UNIQUE 约束会生成隐式索引 sqlite_autoindex_*，
// 这些索引在 sqlite_master 里没有 sql 文本。旧口径从 sqlite_master.sql 正则提取列名，于是：
//   - 元数据把它们报成“无列、非唯一”的假索引（索引管理界面显示空列索引）；
//   - dump 产物生成 `DROP INDEX "sqlite_autoindex_..."` + `CREATE INDEX ... ON t ()`，
//     前者被 sqlite 拒绝（约束索引不可DROP），后者列清单为空；
//     结果：主键不是 INTEGER PRIMARY KEY（rowid别名）的表（如 smallint/text/复合主键），
//     或带内联 UNIQUE 约束的表，其备份产物**必然导入失败**，sqlite 侧结构迁移与备份恢复整体不可用。
//
// 现口径：隐式索引改由 pragma 取列名与唯一性，主键隐式索引不返回（随CREATE TABLE的PRIMARY KEY重建），
// 其余隐式索引在产物中以 idx_ 合法别名重建。本用例把三件事钉死：元数据真实、产物可导入、约束仍生效。
//
// 运行：cd server && go test -tags it -count=1 -run TestITSqliteImplicitIndex ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

const itImplicitIdxTable = "it_impl_idx_tbl"

const itImplicitAutoindexPrefix = "sqlite_autoindex_"

// itPrepareImplicitIndexTable 建含非rowid主键 + 内联UNIQUE约束 + 显式唯一索引的表，并写入若干行
func itPrepareImplicitIndexTable(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	_, err := conn.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, itImplicitIdxTable))
	require.NoError(t, err, "清理用例表失败")
	// id 非 INTEGER PRIMARY KEY（非rowid别名）→ 生成 sqlite_autoindex_*；code 带内联UNIQUE约束
	_, err = conn.Exec(fmt.Sprintf(`CREATE TABLE %s (id SMALLINT PRIMARY KEY, code TEXT UNIQUE, val TEXT, note TEXT)`, itImplicitIdxTable))
	require.NoError(t, err, "建表失败")
	_, err = conn.Exec(fmt.Sprintf(`CREATE UNIQUE INDEX it_idx_named_uq ON %s (val)`, itImplicitIdxTable))
	require.NoError(t, err, "创建显式唯一索引失败")
	_, err = conn.Exec(fmt.Sprintf(`INSERT INTO %s (id, code, val, note) VALUES (1, 'c1', 'v1', 'n1'), (2, 'c2', 'v2', 'n2')`, itImplicitIdxTable))
	require.NoError(t, err, "写入数据失败")
}

// TestITSqliteImplicitIndexMetadata 隐式索引必须如实上报（列名/唯一性），主键隐式索引不进入索引清单
func TestITSqliteImplicitIndexMetadata(t *testing.T) {
	conn := itSqliteNode(t)
	defer conn.Close()
	itPrepareImplicitIndexTable(t, conn)
	defer func() {
		_, _ = conn.Exec("DROP TABLE IF EXISTS " + itImplicitIdxTable)
	}()

	indexs, err := conn.GetMetadata().GetTableIndex(itImplicitIdxTable)
	require.NoError(t, err, "查询索引信息失败")

	byName := make(map[string]dbi.Index, len(indexs))
	for _, idx := range indexs {
		assert.NotEmpty(t, idx.ColumnName, "[%s] 索引列名不得为空（空列会在dump中生成非法DDL）", idx.IndexName)
		assert.NotEqual(t, "id", idx.ColumnName, "主键隐式索引不应进入索引清单（随CREATE TABLE重建，且不可DROP）")
		byName[idx.IndexName] = idx
	}

	// UNIQUE约束生成的隐式索引必须带正确列名与唯一标识
	var codeIdx *dbi.Index
	for name, idx := range byName {
		if strings.HasPrefix(name, itImplicitAutoindexPrefix) {
			codeIdx = &idx
		}
	}
	require.NotNil(t, codeIdx, "内联UNIQUE约束的隐式索引应被上报（否则约束在迁移中静默丢失）")
	assert.Equal(t, "code", codeIdx.ColumnName)
	assert.True(t, codeIdx.IsUnique, "UNIQUE约束索引必须标记为唯一")

	named, ok := byName["it_idx_named_uq"]
	require.True(t, ok, "显式创建的唯一索引必须上报")
	assert.Equal(t, "val", named.ColumnName)
	assert.True(t, named.IsUnique)
}

// TestITSqliteImplicitIndexDumpImport 含隐式索引的表，其dump产物必须可导入且约束仍然生效
func TestITSqliteImplicitIndexDumpImport(t *testing.T) {
	conn := itSqliteNode(t)
	defer conn.Close()
	itPrepareImplicitIndexTable(t, conn)
	defer func() {
		_, _ = conn.Exec("DROP TABLE IF EXISTS " + itImplicitIdxTable)
	}()

	script := itDumpTable(t, conn, itImplicitIdxTable, "sqlite", true, true, "")
	require.Contains(t, strings.ToUpper(script), "CREATE TABLE", "产物应含建表语句，否则本用例失去鉴别力")

	app := &transfer.DbTransferAppImpl{}
	require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)),
		"含隐式索引的sqlite表备份恢复失败\n%s", itTruncate(script, 4000))

	// 主键约束仍在（CREATE TABLE 携带 PRIMARY KEY）
	_, err := conn.Exec(fmt.Sprintf(`INSERT INTO %s (id, code, val, note) VALUES (1, 'c-dup-pk', 'v-dup-pk', 'x')`, itImplicitIdxTable))
	assert.Error(t, err, "导入后主键唯一性丢失（可插入重复id）")

	// 内联UNIQUE约束仍在（隐式索引以合法别名重建）
	_, err = conn.Exec(fmt.Sprintf(`INSERT INTO %s (id, code, val, note) VALUES (99, 'c1', 'v-dup-code', 'x')`, itImplicitIdxTable))
	assert.Error(t, err, "导入后内联UNIQUE约束丢失（可插入重复code）")

	// 显式唯一索引仍在
	_, err = conn.Exec(fmt.Sprintf(`INSERT INTO %s (id, code, val, note) VALUES (98, 'c98', 'v1', 'x')`, itImplicitIdxTable))
	assert.Error(t, err, "导入后显式唯一索引丢失（可插入重复val）")

	// 非冲突数据仍可写入，证明上面的失败源自约束而非表状态异常
	_, err = conn.Exec(fmt.Sprintf(`INSERT INTO %s (id, code, val, note) VALUES (97, 'c97', 'v97', 'ok')`, itImplicitIdxTable))
	require.NoError(t, err, "约束外的正常写入被拒绝")

	_, rows, err := conn.Query(fmt.Sprintf(`SELECT COUNT(*) AS cnt FROM %s`, itImplicitIdxTable))
	require.NoError(t, err)
	cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "行数取值形态异常: %T", rows[0]["cnt"])
	assert.Equal(t, int64(3), cnt, "导入后行数不符（结构段DROP重建后应为源2行+用例1行）")
}
