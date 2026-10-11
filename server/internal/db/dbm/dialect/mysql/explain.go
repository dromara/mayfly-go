package mysql

// mysql 执行计划分析（可选能力 ExplainAnalyzer 的 mysql 实现）。
//
// 用 EXPLAIN <stmt>（不带 ANALYZE）：MySQL 的 EXPLAIN 对 INSERT/UPDATE/DELETE 只做计划分析、
// 不会执行语句，而 EXPLAIN ANALYZE 会真跑一遍——对一条 DELETE 来说那是不可逆的误执行，
// 因此本实现永远不使用 ANALYZE 形态。
//
// 行数取自优化器估算的 rows 列：它是「该步骤预计扫描多少行」，统计信息陈旧或条件无索引时会明显偏离真实值，
// 故 EstimateKind 固定为 explain，上层展示与策略命中原因都必须带上这个口径，不能当成确定值。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

var _ dbi.ExplainAnalyzer = (*MysqlMetadata)(nil)

// mysql 全表/全索引扫描的判据：type=ALL 是全表扫描，type=index 是全索引扫描（两者都不走索引定位）。
// 该判定只在这里出现一次，上层不按 Extra 文本再猜一遍
func mysqlIsFullScan(accessType string) bool {
	switch strings.ToLower(strings.TrimSpace(accessType)) {
	case "all", "index":
		return true
	default:
		return false
	}
}

// Analyze 分析单条语句的执行计划。
//
// 已带 EXPLAIN 前缀的语句直接拒绝：再套一层 EXPLAIN 只会得到与用户预期无关的计划（分析器自身），
// 而不是那条语句的执行计划
func (md *MysqlMetadata) Analyze(ctx context.Context, sql string) (*dbi.ExplainResult, error) {
	stmt := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(sql), ";"))
	if stmt == "" {
		return nil, errorx.NewBiz("sql to analyze is empty")
	}
	if strings.HasPrefix(strings.ToUpper(stmt), "EXPLAIN") || strings.HasPrefix(strings.ToUpper(stmt), "DESC") {
		return nil, errorx.NewBiz("the statement to analyze must not be an EXPLAIN/DESC itself")
	}

	_, rows, err := md.di.QueryContext(ctx, "EXPLAIN "+stmt)
	if err != nil {
		return nil, err
	}

	result := &dbi.ExplainResult{EstimateKind: dbi.EstimateKindExplain}
	seenIndex := make(map[string]bool)
	for _, row := range rows {
		accessType := cast.ToString(row["type"])
		usedIndex := cast.ToString(row["key"])
		fullScan := mysqlIsFullScan(accessType)
		node := dbi.PlanNode{
			NodeKind:     accessType,
			Table:        cast.ToString(row["table"]),
			AccessMethod: accessType,
			IndexName:    usedIndex,
			EstRows:      cast.ToFloat64(row["rows"]),
			FullScan:     fullScan,
		}
		result.Rows += node.EstRows
		if fullScan {
			result.FullScan = true
		}
		if usedIndex != "" && !seenIndex[usedIndex] {
			seenIndex[usedIndex] = true
			result.UsedIndexes = append(result.UsedIndexes, usedIndex)
		}
		result.Plan = append(result.Plan, node)
	}

	// 不填 Children：MySQL 传统 EXPLAIN 是表形态，id 列无法可靠还原父子。
	// 反例：UNION 的三行 id 依次为 1(<union>)、2、3，两个分支都是第 1 行的子节点，
	// 而按「id 更大即内层」逐层挂接会把它们连成 1→2→3 的假链，比不画更坑。
	// 真嵌套关系在 EXPLAIN FORMAT=JSON 的 nested_loop 里，但那是另一套输出形态
	// （版本与 MariaDB 差异大），本轮不拿表形态凑一个树出来
	return result, nil
}
