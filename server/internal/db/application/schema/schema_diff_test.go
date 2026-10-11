package schema

// 结构对比的纯逻辑测试。
//
// 这里钉的全是「不许把话说不准」的情形：跨方言的类型映射产物不是漂移、归类未知不能报成类型不同、
// 内省失败不能报成「一侧缺列」、索引按列组合而非索引名比对、默认值的书写差异不算差异。
// 每条用例的失败形态都是「界面多报或少报一处差异」，而用户会按这些结论去改生产库。

import (
	"testing"

	"mayfly-go/internal/db/dbm/dbi"

	_ "mayfly-go/internal/db/dbm/dialect/mysql"
	_ "mayfly-go/internal/db/dbm/dialect/postgres"
)

func col(name, dataType string, mutators ...func(*dbi.Column)) dbi.Column {
	column := dbi.Column{TableName: "t", ColumnName: name, DataType: dataType, ColumnType: dataType, Nullable: true}
	for _, mutate := range mutators {
		mutate(&column)
	}
	return column
}

func withLength(n int) func(*dbi.Column) { return func(c *dbi.Column) { c.CharMaxLength = n } }
func withScale(p, s int) func(*dbi.Column) {
	return func(c *dbi.Column) { c.NumPrecision, c.NumScale = p, s }
}
func notNull(c *dbi.Column) { c.Nullable = false }
func withDefault(v string) func(*dbi.Column) {
	return func(c *dbi.Column) { c.ColumnDefault = v }
}
func exprDefault(c *dbi.Column) { c.IsExprDefault = true }
func autoInc(c *dbi.Column)     { c.AutoIncrement = true }

func snapshot(dbType string, columns ...dbi.Column) TableSnapshot {
	return TableSnapshot{Name: "t", Columns: columns, DbType: dbType, PrimaryKeysAvailable: true, IndexesAvailable: true}
}

// diffOf 跑一次比对并返回该表的差异（用例都只放一张表）
func diffOf(t *testing.T, src, dst TableSnapshot) *TableDiff {
	t.Helper()
	diffs := DiffSnapshots([]TableSnapshot{src}, []TableSnapshot{dst})
	if len(diffs) != 1 {
		t.Fatalf("expected exactly one table diff entry, got %+v", diffs)
	}
	return diffs[0]
}

func columnDiff(diff *TableDiff, column string) []ColumnDiff {
	got := make([]ColumnDiff, 0, 1)
	for _, item := range diff.Columns {
		if item.Column == column {
			got = append(got, item)
		}
	}
	return got
}

func TestDiffSameStructureReportsNothing(t *testing.T) {
	src := snapshot("mysql", col("id", "bigint"), col("name", "varchar", withLength(50)))
	dst := snapshot("mysql", col("id", "bigint"), col("name", "varchar", withLength(50)))
	if diffs := DiffSnapshots([]TableSnapshot{src}, []TableSnapshot{dst}); len(diffs) != 0 {
		t.Fatalf("identical structures must produce no diff, got %+v", diffs)
	}
}

// TestDiffCrossDialectMappingIsNotDrift 跨方言时，迁移本来就会做的类型转换不算漂移。
//
// mysql 的 blob 迁到 pg 就是 bytea；把它报成类型差异会让每一次跨库对比都满屏假差异，
// 真漂移反而被淹掉。判等只能复用迁移侧的映射规则，不能另立一张等价表
func TestDiffCrossDialectMappingIsNotDrift(t *testing.T) {
	src := snapshot("mysql", col("payload", "blob"), col("flag", "tinyint"), col("at", "datetime"))
	dst := snapshot("postgres", col("payload", "bytea"), col("flag", "int2"), col("at", "timestamp"))

	diffs := DiffSnapshots([]TableSnapshot{src}, []TableSnapshot{dst})
	if len(diffs) != 0 {
		t.Fatalf("mapped types across dialects must not be reported as drift, got %+v", diffs[0])
	}

	// 而映射接不住的就是真差异：mysql tinyint 迁到 pg 是 int2，而不是 bool
	wrong := DiffSnapshots(
		[]TableSnapshot{snapshot("mysql", col("flag", "tinyint"))},
		[]TableSnapshot{snapshot("postgres", col("flag", "bool"))})
	if len(wrong) != 1 || len(wrong[0].Columns) != 1 || wrong[0].Columns[0].Kind != ColumnTypeChanged {
		t.Fatalf("a type the mapping cannot explain must be reported, got %+v", wrong)
	}
}

func TestDiffRealTypeDifferenceIsReported(t *testing.T) {
	src := snapshot("mysql", col("name", "varchar", withLength(50)))
	dst := snapshot("mysql", col("name", "bigint"))

	diff := diffOf(t, src, dst)
	items := columnDiff(diff, "name")
	if len(items) != 1 || items[0].Kind != ColumnTypeChanged {
		t.Fatalf("a genuine type change must be reported, got %+v", diff.Columns)
	}
	// 类型都不同就不该再叠一条长度差异（同一条事实的两种说法）
	for _, item := range items {
		if item.Kind == ColumnSizeChanged {
			t.Fatalf("size must not be compared when the types already differ, got %+v", diff.Columns)
		}
	}
}

// TestDiffUnknownCategoryIsNotTypeChange 类型名认不出来时不得报类型差异，也不能静默判等：
// 必须列入「比不了」。GetDbDataType 对未注册名会静默当成 varchar，拿它去比会把两个根本
// 不同的东西说成相同，而界面上看不到任何痕迹
func TestDiffUnknownCategoryIsNotTypeChange(t *testing.T) {
	src := snapshot("mysql", col("payload", "geometry_collection"))
	dst := snapshot("mysql", col("payload", "varchar", withLength(20)))

	diff := diffOf(t, src, dst)
	for _, item := range diff.Columns {
		if item.Kind == ColumnTypeChanged {
			t.Fatalf("an unresolvable type must not be reported as a type change, got %+v", diff.Columns)
		}
	}
	if !hasAspect(diff, "column-types") {
		t.Fatalf("the unresolvable type must be declared as uncomparable, got %+v", diff.Unavailable)
	}
}

func TestDiffSizeAndNullableAreComparedWithinSameCategory(t *testing.T) {
	src := snapshot("mysql", col("name", "varchar", withLength(50)), col("amount", "decimal", withScale(20, 2)))
	dst := snapshot("mysql", col("name", "varchar", withLength(200)), col("amount", "decimal", withScale(20, 6)), col("extra", "varchar"))

	diff := diffOf(t, src, dst)
	if items := columnDiff(diff, "name"); len(items) != 1 || items[0].Kind != ColumnSizeChanged {
		t.Fatalf("varchar length change must be reported as a size diff, got %+v", diff.Columns)
	}
	if items := columnDiff(diff, "amount"); len(items) != 1 || items[0].Kind != ColumnSizeChanged {
		t.Fatalf("decimal scale change must be reported, got %+v", diff.Columns)
	}
	if items := columnDiff(diff, "extra"); len(items) != 1 || items[0].Kind != ColumnOnlyInTarget {
		t.Fatalf("a column missing on the source side must be reported, got %+v", diff.Columns)
	}
}

// TestDiffLengthUnknownIsNotCompared 0 在这里是「该方言没给出长度」，不是「长度为 0」：
// 拿它判差异会造出一批假漂移，拿它判相等则是在猜
func TestDiffLengthUnknownIsNotCompared(t *testing.T) {
	src := snapshot("mysql", col("name", "varchar"))
	dst := snapshot("mysql", col("name", "varchar", withLength(50)))

	if diffs := DiffSnapshots([]TableSnapshot{src}, []TableSnapshot{dst}); len(diffs) != 0 {
		t.Fatalf("an unavailable length must never be compared, got %+v", diffs[0].Columns)
	}
}

func TestDiffNullableAndAutoIncrement(t *testing.T) {
	src := snapshot("mysql", col("code", "varchar", notNull), col("id", "bigint", autoInc))
	dst := snapshot("mysql", col("code", "varchar"), col("id", "bigint"))

	diff := diffOf(t, src, dst)
	if items := columnDiff(diff, "code"); len(items) != 1 || items[0].Kind != ColumnNullableDiff {
		t.Fatalf("nullable change must be reported, got %+v", diff.Columns)
	}
	if items := columnDiff(diff, "id"); len(items) != 1 || items[0].Kind != ColumnAutoIncDiff {
		t.Fatalf("auto increment change within the same dialect must be reported, got %+v", diff.Columns)
	}

	// 跨方言不比自增：mysql 的 AUTO_INCREMENT 与 pg 的 identity 在元数据里形态不同
	cross := DiffSnapshots(
		[]TableSnapshot{snapshot("mysql", col("id", "bigint", autoInc))},
		[]TableSnapshot{snapshot("postgres", col("id", "int8"))})
	for _, table := range cross {
		for _, item := range table.Columns {
			if item.Kind == ColumnAutoIncDiff {
				t.Fatalf("auto increment must not be compared across dialects, got %+v", table.Columns)
			}
		}
	}
}

// TestDiffDefaultFormsAreNotDrift 默认值的书写形态差异不是漂移：
// 两侧对「无默认值」和「引号包裹」的呈现本来就不一致
func TestDiffDefaultFormsAreNotDrift(t *testing.T) {
	cases := []struct{ src, dst string }{{"", "NULL"}, {"'abc'", "abc"}, {" 0 ", "0"}}
	for _, c := range cases {
		src := snapshot("mysql", col("status", "int", withDefault(c.src)))
		dst := snapshot("mysql", col("status", "int", withDefault(c.dst)))
		diffs := DiffSnapshots([]TableSnapshot{src}, []TableSnapshot{dst})
		if len(diffs) != 0 {
			t.Fatalf("default %q vs %q must not be reported as drift, got %+v", c.src, c.dst, diffs[0].Columns)
		}
	}

	src := snapshot("mysql", col("status", "int", withDefault("1")))
	dst := snapshot("mysql", col("status", "int", withDefault("2")))
	diff := diffOf(t, src, dst)
	if items := columnDiff(diff, "status"); len(items) != 1 || items[0].Kind != ColumnDefaultDiff {
		t.Fatalf("a real default change must be reported, got %+v", diff.Columns)
	}
}

// TestDiffExpressionDefaultIsSkipped 表达式默认值不比对：各方言的函数名与语法互不相通，
// 比出来的一定是「不同」，而那不是漂移（用 text 而不是 uuid：pg 的类型引擎未注册 uuid，
// 那会走「认不出」分支而不是本用例要比的分支）
func TestDiffExpressionDefaultIsSkipped(t *testing.T) {
	src := snapshot("postgres", col("code", "text", withDefault("gen_random_uuid()"), exprDefault))
	dst := snapshot("postgres", col("code", "text"))

	diffs := DiffSnapshots([]TableSnapshot{src}, []TableSnapshot{dst})
	if len(diffs) != 0 {
		t.Fatalf("expression defaults must not be compared, got %+v", diffs[0].Columns)
	}
}

func TestDiffNamesAreCaseInsensitive(t *testing.T) {
	src := snapshot("mysql", dbi.Column{TableName: "T", ColumnName: "Name", DataType: "varchar", ColumnType: "varchar", Nullable: true})
	dst := snapshot("mysql", dbi.Column{TableName: "t", ColumnName: "name", DataType: "varchar", ColumnType: "varchar", Nullable: true})
	diffs := DiffSnapshots([]TableSnapshot{{Name: "T", Columns: src.Columns, DbType: "mysql", PrimaryKeysAvailable: true, IndexesAvailable: true}},
		[]TableSnapshot{{Name: "t", Columns: dst.Columns, DbType: "mysql", PrimaryKeysAvailable: true, IndexesAvailable: true}})
	if len(diffs) != 0 {
		t.Fatalf("table and column names must match case-insensitively, got %+v", diffs)
	}
}

func TestDiffMissingTableBothDirections(t *testing.T) {
	only := TableSnapshot{Name: "z_only", DbType: "mysql"}
	gone := TableSnapshot{Name: "a_gone", DbType: "mysql"}

	diffs := DiffSnapshots([]TableSnapshot{only}, []TableSnapshot{gone})
	if len(diffs) != 2 {
		t.Fatalf("both directions of a missing table must be reported, got %+v", diffs)
	}
	// 表名排序后 a_gone 在前，两个方向各自必须标对 kind
	if diffs[0].Table != "a_gone" || diffs[0].Kind != TableOnlyInTarget {
		t.Fatalf("a table only on the target side must be marked so, got %+v", diffs[0])
	}
	if diffs[1].Table != "z_only" || diffs[1].Kind != TableOnlyInSource {
		t.Fatalf("a table only on the source side must be marked so, got %+v", diffs[1])
	}
}

// TestDiffPrimaryKeyUnknownIsNotADiff 主键读不到与真的没有主键必须分开表达
func TestDiffPrimaryKeyUnknownIsNotADiff(t *testing.T) {
	src := snapshot("mysql", col("id", "bigint"))
	src.PrimaryKeys = []string{"id"}

	// 目标侧没读到主键：不得报成「主键变了」，但必须标出这一面比不了
	unreadable := snapshot("mysql", col("id", "bigint"))
	unreadable.PrimaryKeysAvailable = false
	diff := diffOf(t, src, unreadable)
	if diff.PrimaryKey != nil {
		t.Fatalf("an unreadable primary key must not be reported as a change, got %+v", diff.PrimaryKey)
	}
	if !hasAspect(diff, "primary-key") {
		t.Fatalf("the unreadable side must be listed under unavailable aspects, got %+v", diff.Unavailable)
	}

	// 主键确实不同：必须报出来
	changed := snapshot("mysql", col("id", "bigint"), col("code", "varchar"))
	changed.PrimaryKeys = []string{"code"}
	diff = diffOf(t, src, changed)
	if diff.PrimaryKey == nil || diff.PrimaryKey.Kind == "" {
		t.Fatal("a changed primary key must be reported")
	}

	// 两侧都读到了且都没有主键：合法结构，不算差异也不算比不了
	none := snapshot("mysql", col("id", "bigint"))
	if diffs := DiffSnapshots([]TableSnapshot{none}, []TableSnapshot{none}); len(diffs) != 0 {
		t.Fatalf("both sides having no primary key is not drift, got %+v", diffs)
	}
}

// hasAspect 差异里是否标注了某个「比不了」的方面
func hasAspect(diff *TableDiff, aspect string) bool {
	for _, item := range diff.Unavailable {
		if item.Aspect == aspect {
			return true
		}
	}
	return false
}

// TestDiffIndexesCompareByColumnsNotName 索引按（列组合+顺序）比对：索引名各库自取，按名比会把同一个索引说成两个
func TestDiffIndexesCompareByColumnsNotName(t *testing.T) {
	src := snapshot("mysql", col("a", "int"), col("b", "int"))
	src.Indexes = []dbi.Index{{IndexName: "idx_ab", ColumnName: "a", SeqInIndex: 1}, {IndexName: "idx_ab", ColumnName: "b", SeqInIndex: 2}}

	dst := snapshot("mysql", col("a", "int"), col("b", "int"))
	dst.Indexes = []dbi.Index{{IndexName: "t_a_b_idx", ColumnName: "a", SeqInIndex: 1}, {IndexName: "t_a_b_idx", ColumnName: "b", SeqInIndex: 2}}

	if diffs := DiffSnapshots([]TableSnapshot{src}, []TableSnapshot{dst}); len(diffs) != 0 {
		t.Fatalf("the same columns in the same order under different index names is not drift, got %+v", diffs)
	}

	// 列顺序不同即不同索引：(a,b) 与 (b,a) 的可用性不一样，必须报出来
	reversed := snapshot("mysql", col("a", "int"), col("b", "int"))
	reversed.Indexes = []dbi.Index{{IndexName: "idx_ba", ColumnName: "b", SeqInIndex: 1}, {IndexName: "idx_ba", ColumnName: "a", SeqInIndex: 2}}
	diff := diffOf(t, src, reversed)
	if len(diff.Indexes) != 2 {
		t.Fatalf("index column order matters and must be reported, got %+v", diff.Indexes)
	}

	// 唯一性不同也是差异
	srcOne := snapshot("mysql", col("a", "int"))
	srcOne.Indexes = []dbi.Index{{IndexName: "idx_a", ColumnName: "a", SeqInIndex: 1}}
	dstOne := snapshot("mysql", col("a", "int"))
	dstOne.Indexes = []dbi.Index{{IndexName: "u_a", ColumnName: "a", SeqInIndex: 1, IsUnique: true}}
	diff = diffOf(t, srcOne, dstOne)
	if len(diff.Indexes) != 1 || diff.Indexes[0].Kind != "index-unique-diff" {
		t.Fatalf("a uniqueness change must be reported, got %+v", diff.Indexes)
	}
}

// TestDiffIndexIntrospectionFailureIsDeclared 索引内省失败要标「比不了」，而不是报成「一侧没有索引」
func TestDiffIndexIntrospectionFailureIsDeclared(t *testing.T) {
	src := snapshot("mysql", col("a", "int"))
	src.Indexes = []dbi.Index{{IndexName: "idx_a", ColumnName: "a", SeqInIndex: 1}}
	src.IndexesAvailable = false

	dst := snapshot("mysql", col("a", "int"))

	diff := diffOf(t, src, dst)
	if len(diff.Indexes) != 0 {
		t.Fatalf("an index read failure must not be turned into index diffs, got %+v", diff.Indexes)
	}
	if !hasAspect(diff, "indexes") {
		t.Fatalf("the index read failure must be declared, got %+v", diff.Unavailable)
	}
}

// TestDiffEmptyColumnsIsNotMissingEverywhere 一侧完全读不到列时不能把对面每列都报成「只在一侧存在」：
// 那是内省没拿到数据，不是结构差异
func TestDiffEmptyColumnsIsNotMissingEverywhere(t *testing.T) {
	src := TableSnapshot{Name: "t", DbType: "mysql", PrimaryKeysAvailable: true, IndexesAvailable: true}
	dst := snapshot("mysql", col("a", "int"), col("b", "int"))

	diff := diffOf(t, src, dst)
	for _, item := range diff.Columns {
		if item.Kind == ColumnOnlyInTarget {
			t.Fatalf("unreadable columns must not be reported as one-sided columns, got %+v", diff.Columns)
		}
	}
	found := false
	for _, item := range diff.Unavailable {
		if item.Aspect == "columns" {
			found = true
		}
	}
	if !found {
		t.Fatalf("an empty column read must be declared as unavailable, got %+v", diff.Unavailable)
	}
}
