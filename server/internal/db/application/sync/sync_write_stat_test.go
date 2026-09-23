package sync

import (
	"strings"
	"testing"

	_ "mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bracketQuote 测试用引用器：以 [name] 形式引用标识符，便于断言探测条件文本
func bracketQuote(s string) string { return "[" + s + "]" }

func TestCollectKeyProbeClauses(t *testing.T) {
	rows := []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 1, "name": "b"}, // 同批重复键：计入 keyed，但只产出一个探测条件
		{"id": int64(2), "name": "c"},
		{"id": nil, "name": "d"}, // NULL 冲突键不可能命中唯一约束
		{"name": "e"},            // 冲突键列未映射
	}
	stat := collectKeyProbeClauses(rows, []string{"id"}, bracketQuote, dbi.SQLValueString)
	assert.Equal(t, 5, stat.total)
	assert.Equal(t, 3, stat.keyed)
	assert.Equal(t, []string{"([id] = 1)", "([id] = 2)"}, stat.clauses)
}

func TestCollectKeyProbeClauses_CaseInsensitiveAndComposite(t *testing.T) {
	// 目标列名与行内键大小写不同仍可取值，条件里引用的是元数据列名
	stat := collectKeyProbeClauses([]map[string]any{{"id": 7}}, []string{"ID"}, bracketQuote, dbi.SQLValueString)
	assert.Equal(t, []string{"([ID] = 7)"}, stat.clauses)

	composite := collectKeyProbeClauses([]map[string]any{{"org": "acme", "uid": 7}}, []string{"org", "uid"}, bracketQuote, dbi.SQLValueString)
	assert.Equal(t, []string{"([org] = 'acme' AND [uid] = 7)"}, composite.clauses)

	// 复合键缺一列即无法作为冲突键使用
	partial := collectKeyProbeClauses([]map[string]any{{"org": "acme"}}, []string{"org", "uid"}, bracketQuote, dbi.SQLValueString)
	assert.Zero(t, partial.keyed)
	assert.Empty(t, partial.clauses)
}

func TestKeyProbeSplitUpsert(t *testing.T) {
	tests := []struct {
		name     string
		probe    keyProbe
		existing int
		inserts  int
		updates  int
	}{
		{
			name:     "全部为新增",
			probe:    keyProbe{clauses: []string{"c1", "c2", "c3"}, keyed: 3, total: 3},
			existing: 0, inserts: 3, updates: 0,
		},
		{
			name:     "唯一一行且键已存在",
			probe:    keyProbe{clauses: []string{"c1"}, keyed: 1, total: 1},
			existing: 1, inserts: 0, updates: 1,
		},
		{
			name:     "新增与更新混合",
			probe:    keyProbe{clauses: []string{"c1", "c2"}, keyed: 2, total: 2},
			existing: 1, inserts: 1, updates: 1,
		},
		{
			// 行为 A、A、B 且 A 已存在：A 与其批内重复行都是更新，B 是新增
			name:     "键已存在且批内重复",
			probe:    keyProbe{clauses: []string{"cA", "cB"}, keyed: 3, total: 3},
			existing: 1, inserts: 1, updates: 2,
		},
		{
			// 行为 无键、A、A：无键与 A 各新增一行，A 的重复行命中刚落库的同一行 → 更新
			name:     "键不存在但批内重复",
			probe:    keyProbe{clauses: []string{"cA"}, keyed: 2, total: 3},
			existing: 0, inserts: 2, updates: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inserts, updates := tt.probe.splitUpsert(tt.existing)
			assert.Equal(t, tt.inserts, inserts)
			assert.Equal(t, tt.updates, updates)
			assert.Equal(t, tt.probe.total, inserts+updates, "更新策略下所有行都会落库")
		})
	}
}

func TestKeyProbeSplitIgnore(t *testing.T) {
	tests := []struct {
		name     string
		probe    keyProbe
		existing int
		inserts  int
		ignored  int
	}{
		{
			name: "全部为新增", probe: keyProbe{clauses: []string{"c1", "c2"}, keyed: 2, total: 2},
			existing: 0, inserts: 2, ignored: 0,
		},
		{
			name: "键全部已存在", probe: keyProbe{clauses: []string{"c1", "c2"}, keyed: 2, total: 2},
			existing: 2, inserts: 0, ignored: 2,
		},
		{
			// 行为 A、A、B：A 已存在被忽略，其批内重复行同样被忽略，B 新增
			name: "键已存在且批内重复", probe: keyProbe{clauses: []string{"cA", "cB"}, keyed: 3, total: 3},
			existing: 1, inserts: 1, ignored: 2,
		},
		{
			// 行为 无键、A、A：无键行与首个 A 落库，第二个 A 被忽略
			name: "键不存在但批内重复", probe: keyProbe{clauses: []string{"cA"}, keyed: 2, total: 3},
			existing: 0, inserts: 2, ignored: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inserts, ignored := tt.probe.splitIgnore(tt.existing)
			assert.Equal(t, tt.inserts, inserts)
			assert.Equal(t, tt.ignored, ignored)
			assert.Equal(t, tt.probe.total, inserts+ignored, "忽略策略下每行要么落库要么被忽略")
		})
	}
}

func TestCoarseWriteSplit(t *testing.T) {
	// 探测不可用时的兜底口径与拆分引入前一致：按策略名义整批计数
	assert.Equal(t, writeSplit{updates: 5}, coarseWriteSplit(dbi.DuplicateStrategyUpdate, 5))
	assert.Equal(t, writeSplit{inserts: 5}, coarseWriteSplit(dbi.DuplicateStrategyIgnore, 5))
	assert.Equal(t, writeSplit{inserts: 5}, coarseWriteSplit(dbi.DuplicateStrategyNone, 5))
}

func TestHasUpsertUpdateBranch(t *testing.T) {
	columns := []dbi.Column{{ColumnName: "id"}, {ColumnName: "name"}}
	assert.True(t, hasUpsertUpdateBranch(columns, []string{"id"}), "存在非键列即可生成SET子句")

	// 全列皆为冲突键：方言退化为忽略插入，不得把已存在键的行计为更新
	assert.False(t, hasUpsertUpdateBranch([]dbi.Column{{ColumnName: "id"}}, []string{"id"}))
	assert.False(t, hasUpsertUpdateBranch([]dbi.Column{{ColumnName: "ID"}}, []string{"id"}), "列名大小写不同仍视为同一键列")

	// 非键列但为生成列：同样不可写入
	generated := []dbi.Column{{ColumnName: "id"}, {ColumnName: "total", IsGenerated: true}}
	assert.False(t, hasUpsertUpdateBranch(generated, []string{"id"}))
	assert.True(t, hasUpsertUpdateBranch(append(generated, dbi.Column{ColumnName: "remark"}), []string{"id"}))
}

func TestProbeWriteSplitWithoutProbeQuery(t *testing.T) {
	// 直接插入策略与无冲突键场景都不存在更新分支，整批计新增（且不需要目标连接）
	cases := []struct {
		name     string
		strategy int
		meta     *dbi.TargetTableMeta
	}{
		{"直接插入策略", dbi.DuplicateStrategyNone, &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}},
		{"未配置策略", 0, &dbi.TargetTableMeta{UniqueColumns: []string{"id"}}},
		{"无冲突键", dbi.DuplicateStrategyUpdate, &dbi.TargetTableMeta{}},
		{"未内省目标表元信息", dbi.DuplicateStrategyUpdate, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sec := &syncExecContext{task: &entity.DataSyncTask{DuplicateStrategy: tc.strategy}, targetTableMeta: tc.meta}
			rows := []map[string]any{{"id": 1}, {"id": 2}}
			split, err := probeWriteSplit(t.Context(), sec, rows)
			assert.NoError(t, err)
			assert.Equal(t, writeSplit{inserts: 2}, split)
		})
	}

	empty, err := probeWriteSplit(t.Context(), &syncExecContext{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, writeSplit{}, empty, "空批次不产生任何计数")
}

func TestSyncMetricsRecordWrite(t *testing.T) {
	m := NewSyncMetrics()
	m.RecordWrite(3, 2, 1)
	m.RecordWrite(1, 0, 0)
	assert.Equal(t, 4, m.InsertCount)
	assert.Equal(t, 2, m.UpdateCount)
	assert.Equal(t, 1, m.SkipCount, "被唯一约束忽略的行与过滤/冲突跳过同属未写入")
}

// TestKeyProbeClausesQuotedPerDialect 冲突键条件里的值必须按目标方言转义，
// 否则 MySQL 下含反斜杠的键会探测不到已存在行，把更新误计为新增。
func TestKeyProbeClausesQuotedPerDialect(t *testing.T) {
	rows := []map[string]any{{"id": `a\b`}}
	mysqlStat := collectKeyProbeClauses(rows, []string{"id"}, bracketQuote, dbi.SQLValueStringEscapeBackslash)
	assert.Equal(t, []string{"([id] = 'a\\\\b')"}, mysqlStat.clauses)

	standardStat := collectKeyProbeClauses(rows, []string{"id"}, bracketQuote, dbi.SQLValueString)
	assert.Equal(t, []string{"([id] = 'a\\b')"}, standardStat.clauses)
}

// TestHasUpsertUpdateBranchMatchesDialects 应用层「有无更新分支」的判据必须与方言侧 SET 子句的排除规则同进同退：
// 两边规则一旦漂移（如方言多排除一类列），指标会把未发生的更新计成更新、或把已发生的更新计成新增。
// 断言对象是真实方言生成的语句，故方言改动会在此直接失败，而不是静默产出假数。
func TestHasUpsertUpdateBranchMatchesDialects(t *testing.T) {
	cases := []struct {
		name       string
		columns    []dbi.Column
		uniqueCols []string
		want       bool
	}{
		{"存在可更新的非键列", []dbi.Column{{ColumnName: "id", IsPrimaryKey: true}, {ColumnName: "name"}}, []string{"id"}, true},
		{"全列皆为冲突键", []dbi.Column{{ColumnName: "id", IsPrimaryKey: true}}, []string{"id"}, false},
		{"非键列均为生成列", []dbi.Column{{ColumnName: "id", IsPrimaryKey: true}, {ColumnName: "total", IsGenerated: true}}, []string{"id"}, false},
	}

	// 各方言 upsert 的更新子句标记；无更新分支时方言退化为忽略插入（不含该标记）
	updateMarkers := map[dbi.DbType]string{
		dbi.DbType("mysql"):    "ON DUPLICATE KEY UPDATE",
		dbi.DbType("postgres"): "DO UPDATE SET",
	}

	for dbType, marker := range updateMarkers {
		dialect := dbi.GetDialect(dbType)
		require.NotNil(t, dialect, "方言未注册：%s", dbType)
		gen := dialect.GetSQLGenerator()

		for _, tc := range cases {
			row := make([]any, len(tc.columns))
			for i := range row {
				row[i] = 1
			}
			sqls := gen.GenInsert("t_stat", tc.columns, [][]any{row}, dbi.DuplicateStrategyUpdate, &dbi.TargetTableMeta{UniqueColumns: tc.uniqueCols})
			statement := strings.ToUpper(strings.Join(sqls, " "))

			assert.Equal(t, tc.want, hasUpsertUpdateBranch(tc.columns, tc.uniqueCols), "[%s] 判据不符: %s", tc.name, statement)
			assert.Equal(t, tc.want, strings.Contains(statement, marker), "[%s] 判据与方言实际语句不一致: %s", tc.name, statement)
		}
	}
}
