package shard

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
)

func ident(s string) string { return s } // 不做引用的标识符函数

// plan 用包默认单片行数规划，让各用例只表达「区间与行数」这一层差异
func plan(pk string, pkMin, pkMax int64, totalRows int) []Shard {
	return Plan(pk, pkMin, pkMax, totalRows, 0, ident)
}

// ---------------- DetectIntPrimaryKey ----------------

func TestDetectIntPrimaryKey(t *testing.T) {
	cols := func(defs ...string) []dbi.Column {
		// defs格式: "name:type:pk"（pk可选标记）
		var cs []dbi.Column
		for _, d := range defs {
			parts := strings.Split(d, ":")
			c := dbi.Column{ColumnName: parts[0], DataType: parts[1]}
			if len(parts) > 2 && parts[2] == "pk" {
				c.IsPrimaryKey = true
			}
			cs = append(cs, c)
		}
		return cs
	}

	// 可分片：单列整型主键（各方言整型类型）
	for _, dt := range []string{"int", "bigint", "smallint", "tinyint", "mediumint", "integer", "BIGINT", " int ", "int2", "int4", "int8"} {
		assert.Equal(t, "id", DetectIntPrimaryKey(cols("id:"+dt+":pk", "name:varchar")), "整型类型 %s 应可分片", dt)
	}
	assert.Equal(t, "id", DetectIntPrimaryKey(cols("name:varchar", "id:bigint:pk")))

	// 不可分片矩阵
	assert.Empty(t, DetectIntPrimaryKey(nil), "无列")
	assert.Empty(t, DetectIntPrimaryKey(cols("name:varchar")), "无主键")
	assert.Empty(t, DetectIntPrimaryKey(cols("name:varchar:pk")), "非整型主键")
	assert.Empty(t, DetectIntPrimaryKey(cols("id:decimal(10,0):pk")), "decimal非整型族")
	assert.Empty(t, DetectIntPrimaryKey(cols("id:datetime:pk")), "datetime主键")
	assert.Empty(t, DetectIntPrimaryKey(cols("id:int:pk", "code:bigint:pk")), "联合主键")
}

// ---------------- DetectIntShardKey ----------------

// idx 造一条平铺的索引记录（每列一行，与各方言元数据查询产物同形）
func idx(name, column string, unique bool, seq int) dbi.Index {
	return dbi.Index{IndexName: name, ColumnName: column, IsUnique: unique, SeqInIndex: seq}
}

func TestDetectIntShardKey(t *testing.T) {
	cols := func(defs ...string) []dbi.Column {
		var cs []dbi.Column
		for _, d := range defs {
			parts := strings.Split(d, ":")
			cs = append(cs, dbi.Column{ColumnName: parts[0], DataType: parts[1], IsPrimaryKey: len(parts) > 2 && parts[2] == "pk"})
		}
		return cs
	}

	// 主键优先：即使同时存在唯一索引，也仍用主键（主键在任何方言下都保证非 NULL + 唯一）
	assert.Equal(t, "id", DetectIntShardKey(cols("id:int:pk", "name:varchar", "code:int"),
		[]dbi.Index{idx("uk_code", "code", true, 1)}))

	// 无主键但有单列整型唯一索引：可用（数仓 Unique Key 表、只建 unique 的 id 表）
	assert.Equal(t, "id", DetectIntShardKey(cols("id:bigint", "name:varchar"),
		[]dbi.Index{idx("uk_id", "id", true, 1)}))
	// 列名大小写：返回元数据里的真实写法（上游拼区间谓词与列标签都按它来，比对一律 EqualFold）
	assert.Equal(t, "ID", DetectIntShardKey(cols("ID:bigint", "name:varchar"),
		[]dbi.Index{idx("uk_id", "id", true, 1)}))

	// 不可用矩阵：非唯一索引、联合唯一索引、非整型唯一列、主键索引、无索引
	assert.Empty(t, DetectIntShardKey(cols("id:int"), []dbi.Index{idx("idx_id", "id", false, 1)}), "普通索引不保证唯一")
	assert.Empty(t, DetectIntShardKey(cols("a:int", "b:int"),
		[]dbi.Index{idx("uk_ab", "a", true, 1), idx("uk_ab", "b", true, 2)}), "联合唯一索引的单个分量不唯一")
	assert.Empty(t, DetectIntShardKey(cols("code:varchar"), []dbi.Index{idx("uk_code", "code", true, 1)}), "唯一但非整型")
	assert.Empty(t, DetectIntShardKey(cols("id:int"), []dbi.Index{{IndexName: "PRIMARY", ColumnName: "id", IsUnique: true, IsPrimaryKey: true}}), "主键索引已由 DetectIntPrimaryKey 判定，索引分支不得重复取它")
	assert.Empty(t, DetectIntShardKey(cols("a:int", "b:int"), []dbi.Index{{IndexName: "PRIMARY", ColumnName: "a", IsUnique: true, IsPrimaryKey: true, SeqInIndex: 1}, {IndexName: "PRIMARY", ColumnName: "b", IsUnique: true, IsPrimaryKey: true, SeqInIndex: 2}}), "联合主键既非单列整型主键，也不是可用候选")
	// 缺少列级主键标记时，单列唯一索引就是合法候选（元数据只给了索引侧信息的情形）
	assert.Equal(t, "id", DetectIntShardKey(cols("id:int"), []dbi.Index{idx("uk_id", "id", true, 1)}))
	assert.Empty(t, DetectIntShardKey(cols("id:int"), nil), "无索引元数据")

	// 多个候选时按索引元数据顺序取第一个可用的
	assert.Equal(t, "b", DetectIntShardKey(cols("a:varchar", "b:int"),
		[]dbi.Index{idx("uk_a", "a", true, 1), idx("uk_b", "b", true, 1)}))
}

// ---------------- NormalizeTargetRows ----------------

func TestNormalizeTargetRows(t *testing.T) {
	// 未配置取包默认；越界钳制到上下限（性能旋钮不值得让整轮同步失败）
	assert.Equal(t, TargetRows, NormalizeTargetRows(0))
	assert.Equal(t, TargetRows, NormalizeTargetRows(-1))
	assert.Equal(t, MinTargetRows, NormalizeTargetRows(1), "过小钳到下限")
	assert.Equal(t, MinTargetRows, NormalizeTargetRows(MinTargetRows))
	assert.Equal(t, 50000, NormalizeTargetRows(50000))
	assert.Equal(t, MaxTargetRows, NormalizeTargetRows(MaxTargetRows))
	assert.Equal(t, MaxTargetRows, NormalizeTargetRows(MaxTargetRows+1), "过大钳到上限")
}

func TestPlan_TargetRowsOverridesDefault(t *testing.T) {
	// 单片行数由任务配置驱动：同一区间，25万行按10万/片得3片，按默认20万/片得2片
	assert.Len(t, Plan("id", 1, 250000, 250000, 100000, ident), 3)
	assert.Len(t, plan("id", 1, 250000, 250000), 2)

	// 非法值归一后不报错也不炸出海量分片：1 行/片钳到下限 1000，25 万行 → 250 片
	// （不钳制的话同样区间会切出 25 万片、直接顶到 MaxCount，一轮同步变成上万次往返）
	assert.Len(t, Plan("id", 1, 250000, 250000, 1, ident), 250, "低于下限的单片行数应钳到下限")
}

// ---------------- Plan：边界计算 ----------------

func TestPlan_NotShardable(t *testing.T) {
	assert.Nil(t, plan("", 1, 100, 1000), "无主键列")
	assert.Nil(t, plan("id", 100, 1, 1000), "min>max异常")
}

func TestPlan_SingleShardNoFilter(t *testing.T) {
	// 单分片：无需过滤条件
	assert.Equal(t, []Shard{{Where: ""}}, plan("id", 1, 1, 1))
	assert.Equal(t, []Shard{{Where: ""}}, plan("id", 5, 100, 100))
	assert.Equal(t, []Shard{{Where: ""}}, plan("id", -200, 199, 0))
}

func TestPlan_Boundaries(t *testing.T) {
	// 20万行/片：80万行 → 4片，闭区间无重叠且覆盖整个键空间
	// （首片无下界、末片无上界，见 rangePredicate：规划时刻之后的区间外新行不能被永久漏掉）
	shards := plan("id", 1, 800000, 800000)
	require.Len(t, shards, 4)
	assert.Equal(t, "id <= 200000", shards[0].Where, "首片必须不带下界")
	assert.Equal(t, "id >= 200001 AND id <= 400000", shards[1].Where)
	assert.Equal(t, "id >= 400001 AND id <= 600000", shards[2].Where)
	assert.Equal(t, "id >= 600001", shards[3].Where, "末片必须不带上界")

	// 解析出各片边界并校验连续覆盖（无界端记为 ±inf）
	type rng struct{ lo, hi int64 }
	ranges := make([]rng, 0, len(shards))
	for _, s := range shards {
		lo, hi := parseShardRange(t, s.Where)
		ranges = append(ranges, rng{lo, hi})
	}
	assert.Equal(t, int64(math.MinInt64), ranges[0].lo, "首片必须覆盖到键空间下缘")
	assert.Equal(t, int64(math.MaxInt64), ranges[len(ranges)-1].hi, "末片必须覆盖到键空间上缘")
	for i := 1; i < len(ranges); i++ {
		assert.Equal(t, ranges[i-1].hi+1, ranges[i].lo, "分片应无缝衔接不重叠: 片%d", i)
	}
}

// ---------------- NULL 主键片 ----------------

// 区间谓词永远不匹配 NULL（比较运算遇 NULL 得 UNKNOWN），所以真的存在 NULL 主键行时必须多一片
// `pk IS NULL`，否则那些行会静默不同步（单次大查询读得到它们）；
// 没有 NULL 行时不得追加，否则每次分片都白产一个空片段（sqlite 把未声明 NOT NULL 的列也报成可空）。
func TestWithNullShard(t *testing.T) {
	quote := func(s string) string { return s }
	bounded := []Shard{{Where: "id <= 100"}, {Where: "id >= 101"}}

	assert.Equal(t, bounded, withNullShard("id", bounded, false, quote), "无 NULL 行时不得追加无谓的 NULL 片")

	withNull := withNullShard("id", bounded, true, quote)
	require.Len(t, withNull, 3)
	assert.Equal(t, "id IS NULL", withNull[2].Where, "可空主键必须有一片接住 NULL 行")
	assert.Len(t, bounded, 2, "withNullShard 不得就地改写入参")

	// 整表一片（无谓词）已包含 NULL 行，不再追加；空列表保持空
	assert.Equal(t, []Shard{{Where: ""}}, withNullShard("id", []Shard{{Where: ""}}, true, quote))
	assert.Empty(t, withNullShard("id", nil, true, quote))
	// 引用函数同样作用于 NULL 片，否则带保留字的主键列会拼出非法 SQL
	assert.Contains(t, withNullShard("id", bounded, true, func(s string) string { return "`" + s + "`" })[2].Where, "`id` IS NULL")
}

func TestPkNullable(t *testing.T) {
	cols := []dbi.Column{{ColumnName: "id", Nullable: true}, {ColumnName: "v"}}
	assert.True(t, pkNullable(cols, "id"))
	assert.False(t, pkNullable(cols, "v"))
	assert.False(t, pkNullable(cols, "missing"), "查不到列时不去跑 NULL 探测，也不凭空多一次查询")
}

// DetectIntPrimaryKey 不看可空位：sqlite 等方言把未显式 NOT NULL 的列一律报成可空，
// 拿它拒分片会让整个矩阵静默不分片（恰是分片要治的超时场景）；NULL 掩盖由 PlanTable
// 的探测 + 补片负责（见 pkHasNullRows/withNullShard）。这一条把两个职责的边界钉住。
func TestDetectIntPrimaryKeyIgnoresNullability(t *testing.T) {
	cols := []dbi.Column{{ColumnName: "id", IsPrimaryKey: true, DataType: "bigint", Nullable: true}}
	assert.Equal(t, "id", DetectIntPrimaryKey(cols), "可空主键仍应判为可分片，NULL 行由补片接住")
}

// scanOne 扫出单个整数边界
func scanOne(where string, layout string, out *int64) error {
	var v int64
	n, err := fmt.Sscanf(where, layout, &v)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("where格式异常: %s", where)
	}
	*out = v
	return nil
}

// parseShardRange 解析片谓词为 (lo, hi)，单侧无界用 MinInt64/MaxInt64 表示。
// 三种形态就是 rangePredicate 的输出契约：`id <= hi`（首片）、`id >= lo AND id <= hi`（中间）、
// `id >= lo`（末片）。用例里拿裸列名，引用函数的作用另有专门用例校验。
func parseShardRange(t *testing.T, where string) (int64, int64) {
	t.Helper()
	require.NotEmpty(t, where)
	lo, hi := int64(math.MinInt64), int64(math.MaxInt64)
	if !strings.Contains(where, " AND ") {
		if strings.HasPrefix(where, "id <= ") {
			require.NoError(t, scanOne(where, "id <= %d", &hi))
		} else {
			require.NoError(t, scanOne(where, "id >= %d", &lo))
		}
	} else {
		n, err := fmt.Sscanf(where, "id >= %d AND id <= %d", &lo, &hi)
		require.NoError(t, err)
		require.Equal(t, 2, n, "where格式异常: %s", where)
	}
	require.LessOrEqual(t, lo, hi, "分片边界倒挂: %s", where)
	return lo, hi
}

func TestPlan_Remainder(t *testing.T) {
	// 25万行 → 2片，区间均匀切分（余数均摊）；首/末片的单侧无界即分片布局本身
	shards := plan("id", 1, 250000, 250000)
	require.Len(t, shards, 2)
	assert.Equal(t, "id <= 125000", shards[0].Where)
	assert.Equal(t, "id >= 125001", shards[1].Where)
}

func TestPlan_SparsePkUsesTableRows(t *testing.T) {
	// 稀疏主键（1和10亿，实际2行）：元数据行数优先，不应按区间跨度切出海量分片
	shards := plan("id", 1, 1000000000, 2)
	require.Len(t, shards, 1)
	assert.Equal(t, "", shards[0].Where)
}

func TestPlan_SparsePkWithoutTableRows(t *testing.T) {
	// 无元数据行数时按跨度切分（保守方向：宁可多分片不丢数据）
	shards := plan("id", 1, 1000000000, 0)
	assert.Len(t, shards, 5000)
	// 首尾边界校验：两端各自无界，区间外的新行不会被永久漏掉
	assert.Equal(t, "id <= 200000", shards[0].Where)
	assert.Equal(t, "id >= 999800001", shards[len(shards)-1].Where)
}

func TestPlan_ShardCountCappedByMaxCount(t *testing.T) {
	// 主键区间极大时分片数封顶为MaxCount
	shards := plan("id", 1, 90000000000, 0)
	assert.Len(t, shards, MaxCount)
	assert.Equal(t, "id <= 9000000", shards[0].Where)
	assert.Equal(t, "id >= 89991000001", shards[len(shards)-1].Where)
}

func TestPlan_NegativeAndSmallSpan(t *testing.T) {
	// 负数主键
	shards := plan("id", -100, -1, 100)
	require.Len(t, shards, 1)
	assert.Equal(t, "", shards[0].Where)

	// 区间跨度小于目标行数 → 单片
	assert.Len(t, plan("id", -50, 49, 0), 1)

	// 区间跨度大但行数少（元数据2行）
	shards = plan("id", -1000000, 1000000, 2)
	require.Len(t, shards, 1)
}

func TestPlan_ShardCountCappedBySpan(t *testing.T) {
	// 行数元数据虚高（500万）但主键跨度只 10 → 分片数不超过跨度
	shards := plan("id", 1, 10, 5000000)
	require.Len(t, shards, 10)
	// 每片单值区间，覆盖1~10（首片无下界、末片无上界）
	seen := make(map[int64]bool)
	for i, s := range shards {
		expect := fmt.Sprintf("id >= %d AND id <= %d", i+1, i+1)
		if i == 0 {
			expect = "id <= 1"
		}
		if i == len(shards)-1 {
			expect = "id >= 10"
		}
		assert.Equal(t, expect, s.Where)
		var lo, hi int64
		fmt.Sscanf(s.Where, "id >= %d AND id <= %d", &lo, &hi)
		if i == 0 {
			fmt.Sscanf(s.Where, "id <= %d", &lo)
		}
		if i == len(shards)-1 {
			fmt.Sscanf(s.Where, "id >= %d", &lo)
		}
		seen[lo] = true
		assert.Equal(t, int64(i+1), lo)
	}
	assert.Len(t, seen, 10)
}

func TestPlan_QuoteFunc(t *testing.T) {
	// 引用函数应作用于主键列名（首片无下界，取两片分别验证）
	backtick := func(s string) string { return "`" + s + "`" }
	shards := Plan("id", 1, 250000, 250000, 0, backtick)
	require.Len(t, shards, 2)
	assert.Equal(t, "`id` <= 125000", shards[0].Where)
	assert.Equal(t, "`id` >= 125001", shards[1].Where)
}

// ---------------- 极端值溢出防护 ----------------
func TestPlan_ExtremeInt64Range(t *testing.T) {
	// 全 int64 域：区间幅度用 uint64 运算，`pkMax-pkMin+1` 恰好是 2^64，剩余幅度回绕为 0 →
	// 落进「剩余已被本片容纳」分支，一片覆盖全部（Where 为空即整表扫描）。
	// 与旧实现给出的 `id >= MinInt64 AND id <= MaxInt64` 语义完全等价，都是不丢行的安全结果；
	// 这里钉的是不变量（不得切出会漏区的区间），不是某一句旧文案。
	shards := plan("id", -9223372036854775808, 9223372036854775807, 0)
	require.Len(t, shards, 1)
	assert.Equal(t, "", shards[0].Where, "全域必须退化为无谓词的整表一片，不得切出漏区的区间")

	// 足够大但仍可切分的双域：负主键 + 大区间幅度也要做到片间无缝、且并集覆盖到键空间两端
	shards = plan("id", -4000000000, 4000000000, 0)
	require.Greater(t, len(shards), 2, "8e9 幅度应当切出多片而不是退化为一片")
	prevHi := int64(math.MinInt64)
	for i, s := range shards {
		lo, hi := parseShardRange(t, s.Where)
		if i == 0 {
			assert.Equal(t, int64(math.MinInt64), lo, "首片必须无下界")
		}
		if i > 0 {
			assert.Equal(t, prevHi+1, lo, "片%d与前一档应无缝衔接", i)
		}
		prevHi = hi
	}
	assert.Equal(t, int64(math.MaxInt64), prevHi, "末片必须覆盖到键空间上缘")
}

// TestShardablePkTypesAllConvertibleByValToInt64 分片主键白名单与取值归一化必须闭环。
//
// 缺陷现场：DetectIntPrimaryKey把tinyint/smallint判为可分片，而mssql后端对这两类列的
// MIN/MAX返回int16，value.ValToInt64当时不识别 → 分片规划返回nil → 大表并行迁移静默退化为单任务
func TestShardablePkTypesAllConvertibleByValToInt64(t *testing.T) {
	cols := func(dataType string) []dbi.Column {
		return []dbi.Column{{TableName: "t", ColumnName: "id", DataType: dataType, IsPrimaryKey: true}}
	}
	// 白名单内的每个整型SQL类型，其可能对应的Go窄整型形态都必须可归一
	narrowForms := []any{int8(3), int16(3), int32(3), int64(3), uint8(3), uint16(3), uint32(3), []byte("3"), "3"}
	for _, dt := range []string{"tinyint", "smallint", "mediumint", "int", "integer", "bigint", "int2", "int4", "int8"} {
		assert.Equal(t, "id", DetectIntPrimaryKey(cols(dt)), "整型主键 [%s] 应判为可分片", dt)
		for _, form := range narrowForms {
			got, ok := value.ValToInt64(form)
			assert.True(t, ok, "主键类型[%s]的MIN/MAX返值形态 %T 无法归一，分片会静默退化", dt, form)
			assert.Equal(t, int64(3), got)
		}
	}
}
