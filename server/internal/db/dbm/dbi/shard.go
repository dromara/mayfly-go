package dbi

import (
	"fmt"
	"strings"
)

// ShardTargetRows 单个分片的目标行数（~20万行/片）。
// var形式仅为集成测试可调小以验证多分片路径，生产代码勿修改
var ShardTargetRows = 200000

// ShardMaxCount 单表最大分片数（防止主键区间极大时产生天文数字分片；
// 超出后单片跨度变大，仍能完整覆盖区间）
const ShardMaxCount = 10000

// Shard 表数据分片：按主键范围切分的迁移子任务
type Shard struct {
	// Where 分片过滤条件，如 `id >= 1 AND id <= 200000`；空表示无过滤（整表）
	Where string
}

// DetectIntPrimaryKey 判定表是否存在**真实的单列整型主键**，返回主键列名；否则返回 ""（不应分片）。
//
// 注意：不使用 Metadata.GetPrimaryKey 的"默认第一个字段"兜底——该兜底可能返回非主键字段，
// 若据此生成 min/max 范围过滤，NULL行/不匹配行会被过滤条件排除导致丢数据。
// 仅信任 GetColumns 元数据的 IsPrimaryKey 标识：
//   - 联合主键（多个IsPrimaryKey列）→ 不分片
//   - 无主键 → 不分片
//   - 非整型主键（varchar/复合类型等，无法做数值范围切分）→ 不分片
func DetectIntPrimaryKey(columns []Column) string {
	pk := ""
	pkCount := 0
	for _, c := range columns {
		if !c.IsPrimaryKey {
			continue
		}
		pkCount++
		pk = c.ColumnName
	}
	if pkCount != 1 || !isIntegerDataType(pkDataTypeOf(columns, pk)) {
		return ""
	}
	return pk
}

// pkDataTypeOf 取指定列的数据类型
func pkDataTypeOf(columns []Column, columnName string) string {
	for _, c := range columns {
		if c.ColumnName == columnName {
			return c.DataType
		}
	}
	return ""
}

// isIntegerDataType 判断数据类型是否属于整型族（跨方言常用整型类型的白名单）
func isIntegerDataType(dataType string) bool {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint",
		"int2", "int4", "int8": // pg别名
		return true
	}
	return false
}

// PlanShards 按主键闭区间规划分片：目标每片约ShardTargetRows行，生成
// `pk >= a AND pk <= b` 形式的闭区间过滤条件，分片间无重叠、并集覆盖 [pkMin, pkMax]。
//
// 参数：
//   - pkColumn: 整型主键列名（""返回nil）
//   - pkMin/pkMax: 主键最小/最大值（来自源库MIN/MAX查询；pkMax < pkMin返回nil）
//   - totalRows: 表行数估计（GetTables元数据；<=0时用主键区间跨度估计）
//   - quote: 源库方言标识符引用函数
//
// 返回nil表示不可分片（调用方应整表单任务迁移）；单分片场景Where为空（无需过滤）。
func PlanShards(pkColumn string, pkMin, pkMax int64, totalRows int, quote func(string) string) []Shard {
	if pkColumn == "" || pkMin > pkMax {
		return nil
	}

	// 行数估计：优先元数据行数（对稀疏主键更准确），缺失时用主键区间跨度兜底。
	// 跨度用uint64运算（MaxInt64-MinInt64+1=2^64会溢出uint64，需封顶）
	diff := uint64(pkMax) - uint64(pkMin)
	span := diff
	if diff < ^uint64(0) {
		span = diff + 1
	}
	rowsEstimate := uint64(totalRows)
	if rowsEstimate == 0 {
		rowsEstimate = span
	}

	shardCount := rowsEstimate / uint64(ShardTargetRows)
	if rowsEstimate%uint64(ShardTargetRows) != 0 {
		shardCount++
	}
	if shardCount < 1 {
		shardCount = 1
	}
	// 封顶：分片数不超过主键区间跨度与最大分片数
	if shardCount > span {
		shardCount = span
	}
	if shardCount > ShardMaxCount {
		shardCount = ShardMaxCount
	}
	if shardCount == 1 {
		return []Shard{{Where: ""}}
	}

	// 每片主键区间步长（向上取整，保证覆盖整个区间）
	step := span / shardCount
	if span%shardCount != 0 {
		step++
	}

	shards := make([]Shard, 0, shardCount)
	start := pkMin
	for i := uint64(0); i < shardCount; i++ {
		// end安全计算：步长能被start到pkMax的剩余跨度容纳时才前推，避免int64溢出
		end := pkMax
		if remain := uint64(pkMax) - uint64(start) + 1; remain > step {
			end = start + int64(step) - 1
		}
		shards = append(shards, Shard{
			Where: fmt.Sprintf("%s >= %d AND %s <= %d", quote(pkColumn), start, quote(pkColumn), end),
		})
		if end >= pkMax {
			break
		}
		start = end + 1
	}
	return shards
}
