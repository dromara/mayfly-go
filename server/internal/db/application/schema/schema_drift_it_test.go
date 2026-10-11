//go:build it

package schema

// 结构指纹的真实数据库集成测试。
//
// 单测钉的是指纹算法本身，这里钉的是「从方言读回来的结构事实能不能撑起漂移判定」：
//   - 加一列必须让指纹变（漏报就等于漂移永远不会被通知）；
//   - 只改列注释必须不让指纹变（误报会让通知在几次之后被人关掉）；
//   - 加索引必须让指纹变且数量对得上。
//
// 这三条都依赖各方言元数据查询的真实返回形态（列注释落在哪个字段、索引怎么分组），
// 用假快照测不出来。
//
// 运行：cd server && go test -tags it -count=1 -run TestITSchemaDrift ./internal/db/application/schema/

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

const driftTable = "it_drift_t"

type driftDialect struct {
	dbType dbi.DbType
	// create 建表：id 主键 + 一个带注释的文本列
	create string
	// addColumn 给表加一列
	addColumn string
	// onlyComment 只改列注释，不动类型/可空/默认值
	onlyComment string
	// addIndex 给新列加一个索引
	addIndex string
}

func driftDialects() []driftDialect {
	return []driftDialect{
		{
			dbType:      "mysql",
			create:      "CREATE TABLE %s (id INT PRIMARY KEY, name VARCHAR(20) COMMENT '姓名')",
			addColumn:   "ALTER TABLE %s ADD COLUMN age INT",
			onlyComment: "ALTER TABLE %s MODIFY COLUMN name VARCHAR(20) COMMENT '客户姓名'",
			addIndex:    "CREATE INDEX idx_age ON %s (age)",
		},
		{
			dbType:      "postgres",
			create:      `CREATE TABLE %s (id INT PRIMARY KEY, name VARCHAR(20))`,
			addColumn:   "ALTER TABLE %s ADD COLUMN age INT",
			onlyComment: `COMMENT ON COLUMN %s.name IS '客户姓名'`,
			addIndex:    "CREATE INDEX idx_age ON %s (age)",
		},
	}
}

// dropDriftTable 用例表用完即删：临时库虽会随进程销毁，同一轮里多个用例复用同一名表会互相干扰
func dropDriftTable(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	exec(t, conn, "DROP TABLE IF EXISTS "+conn.GetDialect().Quoter().QuoteIdent(driftTable))
}

// driftSnapshot 取漂移用例表的快照摘要；表读不到时直接失败（说明采集链路断了）
func driftSnapshot(t *testing.T, conn *dbi.DbConn) TableSummary {
	t.Helper()
	snapshots, err := loadSnapshots(context.Background(), conn, []string{driftTable})
	require.NoError(t, err)
	require.Len(t, snapshots, 1, "漂移用例表必须能被采集到")
	return SummarizeTable(snapshots[0])
}

func TestITSchemaDriftFingerprint(t *testing.T) {
	for _, dialect := range driftDialects() {
		dialect := dialect
		t.Run(string(dialect.dbType), func(t *testing.T) {
			conn := mustConn(t, dialect.dbType, "drift-fp")
			defer conn.Close()

			quote := conn.GetDialect().Quoter().QuoteIdent
			dropDriftTable(t, conn)
			exec(t, conn, fmt.Sprintf(dialect.create, quote(driftTable)))
			defer dropDriftTable(t, conn)

			base := driftSnapshot(t, conn)
			require.NotEmpty(t, base.Fingerprint)
			assert.Equal(t, 2, base.ColumnCount, "建表后应是两列")
			assert.Equal(t, "id", base.PrimaryKeys, "主键必须被真实采集到")

			// 只改注释：漂移判定不该被噪声触发
			exec(t, conn, fmt.Sprintf(dialect.onlyComment, quote(driftTable)))
			commented := driftSnapshot(t, conn)
			assert.Equal(t, base.Fingerprint, commented.Fingerprint, "改列注释不得算作结构漂移")

			// 加一列：必须被感知，且明细能说出列数从几变到几
			exec(t, conn, fmt.Sprintf(dialect.addColumn, quote(driftTable)))
			afterAdd := driftSnapshot(t, conn)
			assert.NotEqual(t, base.Fingerprint, afterAdd.Fingerprint, "新增列必须被指纹感知")
			changes := DriftChanges(base, afterAdd)
			require.Len(t, changes, 1)
			assert.Equal(t, DriftKindColumnCount, changes[0].Kind)
			assert.Equal(t, "2", changes[0].From)
			assert.Equal(t, "3", changes[0].To)

			// 加索引：同样必须被感知
			exec(t, conn, fmt.Sprintf(dialect.addIndex, quote(driftTable)))
			withIndex := driftSnapshot(t, conn)
			assert.NotEqual(t, afterAdd.Fingerprint, withIndex.Fingerprint, "新增索引必须被指纹感知")
			assert.Equal(t, 1, withIndex.IndexCount, "索引数量应按列组合归并后为 1，实际：%+v", withIndex)
			indexChanges := DriftChanges(afterAdd, withIndex)
			require.Len(t, indexChanges, 1)
			assert.Equal(t, DriftKindIndexCount, indexChanges[0].Kind)
		})
	}
}

// TestITSchemaDriftPrimaryKeyChange 换主键必须被感知：主键是回滚定位与冲突判定的依据，
// 它变了却不报，之后按旧主键生成的语句会打到错的行集上
func TestITSchemaDriftPrimaryKeyChange(t *testing.T) {
	for _, dialect := range driftDialects() {
		dialect := dialect
		t.Run(string(dialect.dbType), func(t *testing.T) {
			conn := mustConn(t, dialect.dbType, "drift-pk")
			defer conn.Close()

			quote := conn.GetDialect().Quoter().QuoteIdent
			dropDriftTable(t, conn)
			exec(t, conn, fmt.Sprintf(dialect.create, quote(driftTable)))
			defer dropDriftTable(t, conn)

			before := driftSnapshot(t, conn)
			require.True(t, before.PrimaryKeysAvailable)
			require.Equal(t, "id", before.PrimaryKeys)

			// 两方言都是「先删旧主键约束再建联合主键」
			switch dialect.dbType {
			case "mysql":
				exec(t, conn, "ALTER TABLE "+quote(driftTable)+" DROP PRIMARY KEY, ADD PRIMARY KEY (id, name)")
			case "postgres":
				exec(t, conn, "ALTER TABLE "+quote(driftTable)+" DROP CONSTRAINT "+quote(driftTable+"_pkey")+", ADD PRIMARY KEY (id, name)")
			}

			after := driftSnapshot(t, conn)
			assert.NotEqual(t, before.Fingerprint, after.Fingerprint, "主键变化必须被指纹感知")
			assert.Equal(t, "id,name", after.PrimaryKeys, "联合主键的列序也必须被如实采集，实际：%+v", after)

			changes := DriftChanges(before, after)
			require.Len(t, changes, 1)
			assert.Equal(t, DriftKindPrimaryKeys, changes[0].Kind)
			assert.Equal(t, "id", changes[0].From)
			assert.Equal(t, "id,name", changes[0].To)
		})
	}
}
