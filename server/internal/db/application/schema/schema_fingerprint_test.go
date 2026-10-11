package schema

// 结构指纹的守卫。
//
// 指纹是漂移检测唯一的判据，两条失效方向都要堵：
//   - 该变的没变（漏报）：结构已被改，巡检却说一致，用户永远收不到通知；
//   - 不该变的乱变（误报）：改个注释、索引被自动改名就报漂移，几次之后通知就被关掉。
//
// 因此除了逐项敏感性，这里还钉住「与差异比对同口径」：注释与索引名不参与比对，也就不参与指纹。

import (
	"testing"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fpColumn(name, columnType string, overrides ...func(*dbi.Column)) dbi.Column {
	column := dbi.Column{TableName: "t1", ColumnName: name, ColumnType: columnType, DataType: columnType}
	for _, override := range overrides {
		override(&column)
	}
	return column
}

func fpTable(columns ...dbi.Column) TableSnapshot {
	return TableSnapshot{
		Name:                 "t1",
		Columns:              columns,
		PrimaryKeys:          []string{"id"},
		PrimaryKeysAvailable: true,
		IndexesAvailable:     true,
		DbType:               "mysql",
	}
}

func fingerprintOf(table TableSnapshot) string {
	return SummarizeTable(table).Fingerprint
}

func TestFingerprintIsSensitiveToStructure(t *testing.T) {
	base := fingerprintOf(fpTable(
		fpColumn("id", "bigint"),
		fpColumn("name", "varchar(20)"),
	))

	cases := []struct {
		name  string
		table TableSnapshot
	}{
		{"新增列", fpTable(fpColumn("id", "bigint"), fpColumn("name", "varchar(20)"), fpColumn("age", "int"))},
		{"删除列", fpTable(fpColumn("id", "bigint"))},
		{"列类型变化", fpTable(fpColumn("id", "bigint"), fpColumn("name", "varchar(30)"))},
		{"列改为可空", fpTable(fpColumn("id", "bigint"), fpColumn("name", "varchar(20)", func(c *dbi.Column) { c.Nullable = true }))},
		{"默认值变化", fpTable(fpColumn("id", "bigint"), fpColumn("name", "varchar(20)", func(c *dbi.Column) { c.ColumnDefault = "'x'" }))},
		{"自增变化", fpTable(fpColumn("id", "bigint", func(c *dbi.Column) { c.AutoIncrement = true }), fpColumn("name", "varchar(20)"))},
		// 列顺序是结构的一部分（ALTER TABLE ... AFTER 会改它），不能因为集合相同就算一致
		{"列顺序变化", fpTable(fpColumn("name", "varchar(20)"), fpColumn("id", "bigint"))},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEqual(t, base, fingerprintOf(tc.table), "%s 必须被指纹感知", tc.name)
		})
	}
}

func TestFingerprintIgnoresNonStructuralNoise(t *testing.T) {
	withComment := fpTable(fpColumn("id", "bigint"), fpColumn("name", "varchar(20)", func(c *dbi.Column) { c.ColumnComment = "姓名" }))
	withoutComment := fpTable(fpColumn("id", "bigint"), fpColumn("name", "varchar(20)"))
	// 差异比对不比注释，指纹也不比：否则改一句注释就推一条漂移通知
	assert.Equal(t, fingerprintOf(withoutComment), fingerprintOf(withComment), "列注释变化不算结构漂移")

	// 索引名是各库自取的（mysql 与 pg 命名规则不同），按列组合归并后名字不参与
	base := fpTable(fpColumn("id", "bigint"))
	base.Indexes = []dbi.Index{{IndexName: "idx_a", ColumnName: "name", IsUnique: true}}
	renamed := fpTable(fpColumn("id", "bigint"))
	renamed.Indexes = []dbi.Index{{IndexName: "t1_name_idx", ColumnName: "name", IsUnique: true}}
	assert.Equal(t, fingerprintOf(base), fingerprintOf(renamed), "索引改名不算结构漂移")

	// 同一个列组合上多一个非唯一索引：数量变了，必须算漂移
	added := fpTable(fpColumn("id", "bigint"))
	added.Indexes = []dbi.Index{
		{IndexName: "idx_a", ColumnName: "name", IsUnique: true},
		{IndexName: "idx_b", ColumnName: "age"},
	}
	assert.NotEqual(t, fingerprintOf(base), fingerprintOf(added), "索引数量变化必须被感知")

	uniqueChanged := fpTable(fpColumn("id", "bigint"))
	uniqueChanged.Indexes = []dbi.Index{{IndexName: "idx_a", ColumnName: "name"}}
	assert.NotEqual(t, fingerprintOf(base), fingerprintOf(uniqueChanged), "索引唯一性变化必须被感知")
}

func TestFingerprintKeepsUnavailableApartFromEmpty(t *testing.T) {
	// 主键「读不到」与「确实没有主键」是两回事，哈希成同一个值就会把一次权限受限的内省记成基线
	notRead := fpTable(fpColumn("id", "bigint"))
	notRead.PrimaryKeysAvailable = false
	noPk := fpTable(fpColumn("id", "bigint"))
	noPk.PrimaryKeys = nil
	assert.NotEqual(t, fingerprintOf(noPk), fingerprintOf(notRead), "主键读不到不得等同于无主键")

	indexesNotRead := fpTable(fpColumn("id", "bigint"))
	indexesNotRead.IndexesAvailable = false
	noIndexes := fpTable(fpColumn("id", "bigint"))
	assert.NotEqual(t, fingerprintOf(noIndexes), fingerprintOf(indexesNotRead), "索引读不到不得等同于无索引")
}

func TestDriftChanges(t *testing.T) {
	baseline := TableSummary{ColumnCount: 8, PrimaryKeys: "id", PrimaryKeysAvailable: true, IndexCount: 2, IndexesAvailable: true}

	columnAdded := baseline
	columnAdded.ColumnCount = 9
	columnAdded.IndexCount = 3
	pkChanged := baseline
	pkChanged.PrimaryKeys = "id,tenant_id"

	changes := DriftChanges(baseline, columnAdded)
	require.Len(t, changes, 2)
	assert.Equal(t, DriftKindColumnCount, changes[0].Kind)
	assert.Equal(t, "8", changes[0].From)
	assert.Equal(t, "9", changes[0].To)
	assert.Equal(t, DriftKindIndexCount, changes[1].Kind)

	pkChanges := DriftChanges(baseline, pkChanged)
	require.Len(t, pkChanges, 1)
	assert.Equal(t, DriftKindPrimaryKeys, pkChanges[0].Kind)
	assert.Equal(t, "id,tenant_id", pkChanges[0].To)

	// 指纹变了但摘要全都没变：必须归因到列定义/列顺序，不能返回空列表（那等于说没变）
	fallback := DriftChanges(baseline, baseline)
	require.Len(t, fallback, 1)
	assert.Equal(t, DriftKindColumnDefs, fallback[0].Kind)
}
