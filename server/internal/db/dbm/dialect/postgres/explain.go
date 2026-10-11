package postgres

// postgres 执行计划分析（可选能力 ExplainAnalyzer 的 postgres 实现）。
//
// 用 EXPLAIN (FORMAT JSON)：结构化输出可直接解析出节点类型、估算行数与所用索引，
// 比解析文本形态稳定（文本格式随版本变列宽/换行）。不带 ANALYZE，因此 UPDATE/DELETE 只被分析不被执行。
//
// 行数取根节点的 "Plan Rows"（优化器估算）。它与 mysql 一样会因统计信息陈旧而偏离真实值，
// 故上层展示与策略命中原因都要带 EstimateKind 说明这是估算值。

import (
	"context"
	"encoding/json"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

var _ dbi.ExplainAnalyzer = (*PgsqlMetadata)(nil)

// explainPlanNode pg 计划节点的 JSON 形态（只声明本实现要用的字段）
type explainPlanNode struct {
	NodeType           string            `json:"Node Type"`
	RelationName       string            `json:"Relation Name"`
	IndexName          string            `json:"Index Name"`
	ParentRelationship string            `json:"Parent Relationship"`
	PlanRows           float64           `json:"Plan Rows"`
	TotalCost          float64           `json:"Total Cost"`
	Plans              []explainPlanNode `json:"Plans"`
}

// explainOutput EXPLAIN (FORMAT JSON) 的顶层形态：单元素数组，元素里一个 Plan 根节点
type explainOutput struct {
	Plan explainPlanNode `json:"Plan"`
}

// pgIsFullScan 顺序扫描即全表扫描（pg 里 Node Type = Seq Scan）
func pgIsFullScan(nodeType string) bool {
	return strings.EqualFold(strings.TrimSpace(nodeType), "Seq Scan")
}

func (pm *PgsqlMetadata) Analyze(ctx context.Context, sql string) (*dbi.ExplainResult, error) {
	stmt := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(sql), ";"))
	if stmt == "" {
		return nil, errorx.NewBiz("sql to analyze is empty")
	}
	if strings.HasPrefix(strings.ToUpper(stmt), "EXPLAIN") {
		return nil, errorx.NewBiz("the statement to analyze must not be an EXPLAIN itself")
	}

	_, rows, err := pm.di.QueryContext(ctx, "EXPLAIN (FORMAT JSON) "+stmt)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errorx.NewBiz("explain returned no plan")
	}

	// 结果只有一列（列名随版本为 QUERY PLAN / QUERY PLAN 别名），取其文本再解析
	raw := rows[0]
	var planText string
	for _, v := range raw {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			planText = s
			break
		}
		planText = cast.ToString(v)
		if strings.TrimSpace(planText) != "" {
			break
		}
	}
	if strings.TrimSpace(planText) == "" {
		return nil, errorx.NewBiz("explain returned an empty plan text")
	}

	var output []explainOutput
	if err := json.Unmarshal([]byte(planText), &output); err != nil {
		return nil, errorx.NewBizf("failed to parse the explain plan: %s", err.Error())
	}
	if len(output) == 0 {
		return nil, errorx.NewBiz("explain returned no plan node")
	}

	result := &dbi.ExplainResult{
		EstimateKind: dbi.EstimateKindExplain,
		Rows:         output[0].Plan.PlanRows,
	}
	collectPgPlanNodes(output[0].Plan, result, make(map[string]bool))
	return result, nil
}

// collectPgPlanNodes 前序递归展开计划节点：标记是否存在顺序扫描、收集用到的索引，
// 并把子节点在本方言里真实给出的嵌套关系（JSON 的 Plans 数组）回填进 Children。
// 返回本节点在 result.Plan 里的下标，供父节点登记子边
func collectPgPlanNodes(node explainPlanNode, result *dbi.ExplainResult, seenIndex map[string]bool) int {
	fullScan := pgIsFullScan(node.NodeType)
	if fullScan {
		result.FullScan = true
	}
	if node.IndexName != "" && !seenIndex[node.IndexName] {
		seenIndex[node.IndexName] = true
		result.UsedIndexes = append(result.UsedIndexes, node.IndexName)
	}
	self := len(result.Plan)
	result.Plan = append(result.Plan, dbi.PlanNode{
		NodeKind:     node.NodeType,
		Table:        node.RelationName,
		AccessMethod: node.NodeType,
		IndexName:    node.IndexName,
		EstRows:      node.PlanRows,
		FullScan:     fullScan,
	})
	for _, child := range node.Plans {
		result.Plan[self].Children = append(result.Plan[self].Children, collectPgPlanNodes(child, result, seenIndex))
	}
	return self
}
