package dbi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ident(s string) string { return s } // 不做引用的标识符函数

// ---------------- DetectIntPrimaryKey ----------------

func TestDetectIntPrimaryKey(t *testing.T) {
	cols := func(defs ...string) []Column {
		// defs格式: "name:type:pk"（pk可选标记）
		var cs []Column
		for _, d := range defs {
			parts := strings.Split(d, ":")
			c := Column{ColumnName: parts[0], DataType: parts[1]}
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

// ---------------- PlanShards：边界计算 ----------------

func TestPlanShards_NotShardable(t *testing.T) {
	assert.Nil(t, PlanShards("", 1, 100, 1000, ident), "无主键列")
	assert.Nil(t, PlanShards("id", 100, 1, 1000, ident), "min>max异常")
}

func TestPlanShards_SingleShardNoFilter(t *testing.T) {
	// 单分片：无需过滤条件
	assert.Equal(t, []Shard{{Where: ""}}, PlanShards("id", 1, 1, 1, ident))
	assert.Equal(t, []Shard{{Where: ""}}, PlanShards("id", 5, 100, 100, ident))
	assert.Equal(t, []Shard{{Where: ""}}, PlanShards("id", -200, 199, 0, ident))
}

func TestPlanShards_Boundaries(t *testing.T) {
	// 20万行/片：80万行 → 4片，闭区间无重叠且覆盖[min,max]
	shards := PlanShards("id", 1, 800000, 800000, ident)
	require.Len(t, shards, 4)

	// 解析出各片边界并校验连续覆盖
	type rng struct{ lo, hi int64 }
	ranges := make([]rng, 0, len(shards))
	for _, s := range shards {
		var lo, hi int64
		n, err := fmt.Sscanf(s.Where, "id >= %d AND id <= %d", &lo, &hi)
		require.NoError(t, err)
		require.Equal(t, 2, n, "where格式异常: %s", s.Where)
		require.LessOrEqual(t, lo, hi, "分片边界倒挂: %s", s.Where)
		ranges = append(ranges, rng{lo, hi})
	}
	assert.Equal(t, int64(1), ranges[0].lo, "首片从min开始")
	assert.Equal(t, int64(800000), ranges[len(ranges)-1].hi, "末片到max结束")
	for i := 1; i < len(ranges); i++ {
		assert.Equal(t, ranges[i-1].hi+1, ranges[i].lo, "分片应无缝衔接不重叠: 片%d", i)
	}
}

func TestPlanShards_Remainder(t *testing.T) {
	// 25万行 → 2片，区间均匀切分（余数均摊）
	shards := PlanShards("id", 1, 250000, 250000, ident)
	require.Len(t, shards, 2)
	assert.Equal(t, "id >= 1 AND id <= 125000", shards[0].Where)
	assert.Equal(t, "id >= 125001 AND id <= 250000", shards[1].Where)
}

func TestPlanShards_SparsePkUsesTableRows(t *testing.T) {
	// 稀疏主键（1和10亿，实际2行）：元数据行数优先，不应按区间跨度切出海量分片
	shards := PlanShards("id", 1, 1000000000, 2, ident)
	require.Len(t, shards, 1)
	assert.Equal(t, "", shards[0].Where)
}

func TestPlanShards_SparsePkWithoutTableRows(t *testing.T) {
	// 无元数据行数时按跨度切分（保守方向：宁可多分片不丢数据）
	shards := PlanShards("id", 1, 1000000000, 0, ident)
	assert.Len(t, shards, 5000)
	// 首尾边界校验
	assert.Equal(t, "id >= 1 AND id <= 200000", shards[0].Where)
	assert.Equal(t, "id >= 999800001 AND id <= 1000000000", shards[len(shards)-1].Where)
}

func TestPlanShards_ShardCountCappedByMaxCount(t *testing.T) {
	// 主键区间极大时分片数封顶为ShardMaxCount
	shards := PlanShards("id", 1, 90000000000, 0, ident)
	assert.Len(t, shards, ShardMaxCount)
	assert.Equal(t, "id >= 1 AND id <= 9000000", shards[0].Where)
	assert.Equal(t, "id >= 89991000001 AND id <= 90000000000", shards[len(shards)-1].Where)
}

func TestPlanShards_NegativeAndSmallSpan(t *testing.T) {
	// 负数主键
	shards := PlanShards("id", -100, -1, 100, ident)
	require.Len(t, shards, 1)
	assert.Equal(t, "", shards[0].Where)

	// 区间跨度小于目标行数 → 单片
	assert.Len(t, PlanShards("id", -50, 49, 0, ident), 1)

	// 区间跨度大但行数少（元数据2行）
	shards = PlanShards("id", -1000000, 1000000, 2, ident)
	require.Len(t, shards, 1)
}

func TestPlanShards_ShardCountCappedBySpan(t *testing.T) {
	// 行数元数据虚高（500万）但主键跨度只有10 → 分片数不超过跨度
	shards := PlanShards("id", 1, 10, 5000000, ident)
	require.Len(t, shards, 10)
	// 每片单值区间，覆盖1~10
	seen := make(map[int64]bool)
	for i, s := range shards {
		expect := fmt.Sprintf("id >= %d AND id <= %d", i+1, i+1)
		assert.Equal(t, expect, s.Where)
		var lo, hi int64
		fmt.Sscanf(s.Where, "id >= %d AND id <= %d", &lo, &hi)
		seen[lo] = true
		assert.Equal(t, int64(i+1), lo)
	}
	assert.Len(t, seen, 10)
}

func TestPlanShards_QuoteFunc(t *testing.T) {
	// 引用函数应作用于主键列名
	backtick := func(s string) string { return "`" + s + "`" }
	shards := PlanShards("id", 1, 250000, 250000, backtick)
	require.Len(t, shards, 2)
	assert.Equal(t, "`id` >= 1 AND `id` <= 125000", shards[0].Where)
}

// ---------------- 极端值溢出防护 ----------------
func TestPlanShards_ExtremeInt64Range(t *testing.T) {
	// 全int64域：uint64跨度运算不得溢出导致错误分片
	shards := PlanShards("id", -9223372036854775808, 9223372036854775807, 0, ident)
	require.NotEmpty(t, shards)
	// 首片下界 = MinInt64
	assert.Contains(t, shards[0].Where, ">= -9223372036854775808")
	// 末片上界 = MaxInt64
	last := shards[len(shards)-1].Where
	assert.True(t, strings.HasSuffix(last, "<= 9223372036854775807"), "末片应覆盖到MaxInt64: %s", last)

	// 分片衔接性校验（解析全部边界确保无重叠、无缝隙）
	prevHi := int64(-9223372036854775808)
	first := true
	for i, s := range shards {
		var lo, hi int64
		_, err := fmt.Sscanf(s.Where, "id >= %d AND id <= %d", &lo, &hi)
		require.NoError(t, err, "片%d: %s", i, s.Where)
		if !first {
			assert.Equal(t, prevHi+1, lo, "片%d与前一档应无缝衔接", i)
		}
		prevHi = hi
		first = false
	}
}
