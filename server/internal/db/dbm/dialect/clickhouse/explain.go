package clickhouse

// clickhouse 执行计划分析（可选能力 ExplainAnalyzer 的 clickhouse 实现）。
//
// 用 EXPLAIN json=1, estimates=1：结构化 JSON 计划树，带每节点行数估算（EstimatedRows），比解析文本形态稳定。
// EXPLAIN 只分析不执行（clickhouse 的 EXPLAIN 本就不执行被分析语句）。
//
// 全表扫描判据：MergeTree 读取节点未用到主键/索引（无 KeyDescription/IndexName）即视为全量读；
// ReadFromStorage 之类无索引的存储读取同样计全表扫描。clickhouse 无「聚簇索引 seek」这种关系型语义，
// 故本实现只如实标注能确定的事实，不套用关系型方言的 seek 判定。

import (
	"context"
	"encoding/json"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

var _ dbi.ExplainAnalyzer = (*ClickHouseMetadata)(nil)

// chPlanNode clickhouse EXPLAIN json 的一个计划节点（只声明用到的字段）。
type chPlanNode struct {
	NodeName       string       `json:"NodeName"`
	Table          string       `json:"Table"`
	IndexName      string       `json:"IndexName"`
	KeyDescription string       `json:"KeyDescription"`
	EstimatedRows  *float64     `json:"EstimatedRows"`
	Nodes          []chPlanNode `json:"Nodes"`
}

// chIsFullScan clickhouse 读取节点未命中主键/索引即全量扫描。
func chIsFullScan(n chPlanNode) bool {
	if n.NodeName == "ReadFromStorage" {
		return true
	}
	if n.NodeName == "ReadFromMergeTree" {
		return n.KeyDescription == "" && n.IndexName == ""
	}
	return false
}

func (cm *ClickHouseMetadata) Analyze(ctx context.Context, sql string) (*dbi.ExplainResult, error) {
	stmt := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(sql), ";"))
	if stmt == "" {
		return nil, errorx.NewBiz("sql to analyze is empty")
	}
	upper := strings.ToUpper(stmt)
	if strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "DESCRIBE") {
		return nil, errorx.NewBiz("the statement to analyze must not be an EXPLAIN/DESCRIBE itself")
	}

	_, rows, err := cm.di.QueryContext(ctx, "EXPLAIN json=1, estimates=1 "+stmt)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errorx.NewBiz("explain returned no plan")
	}

	// 结果单行单列（列名 explain），值为计划 JSON 文本
	var planText string
	for _, v := range rows[0] {
		if s := cast.ToString(v); strings.Contains(s, "\"Plan\"") {
			planText = s
			break
		}
	}
	if planText == "" {
		return nil, errorx.NewBiz("explain returned no plan json")
	}

	root, err := parseChPlanRoot(planText)
	if err != nil {
		return nil, err
	}

	result := &dbi.ExplainResult{EstimateKind: dbi.EstimateKindExplain}
	if root.EstimatedRows != nil {
		result.Rows = *root.EstimatedRows
	}
	collectChPlanNodes(*root, result, make(map[string]bool))
	return result, nil
}

// parseChPlanRoot 兼容 EXPLAIN json 顶层为对象 {"Plan":...} 或单元素数组 [{"Plan":...}] 两种形态。
func parseChPlanRoot(planText string) (*chPlanNode, error) {
	var wrapper struct {
		Plan chPlanNode `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(planText), &wrapper); err == nil && wrapper.Plan.NodeName != "" {
		return &wrapper.Plan, nil
	}
	var arr []struct {
		Plan chPlanNode `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(planText), &arr); err == nil && len(arr) > 0 && arr[0].Plan.NodeName != "" {
		return &arr[0].Plan, nil
	}
	return nil, errorx.NewBiz("failed to parse the explain plan json")
}

// collectChPlanNodes 前序展开计划树：登记节点、判全表扫描、收集索引名，并把 Nodes 嵌套回填成父子边。
func collectChPlanNodes(node chPlanNode, result *dbi.ExplainResult, seenIndex map[string]bool) int {
	fullScan := chIsFullScan(node)
	if fullScan {
		result.FullScan = true
	}
	if node.IndexName != "" && !seenIndex[node.IndexName] {
		seenIndex[node.IndexName] = true
		result.UsedIndexes = append(result.UsedIndexes, node.IndexName)
	}
	est := 0.0
	if node.EstimatedRows != nil {
		est = *node.EstimatedRows
	}
	self := len(result.Plan)
	result.Plan = append(result.Plan, dbi.PlanNode{
		NodeKind:     node.NodeName,
		Table:        node.Table,
		AccessMethod: node.NodeName,
		IndexName:    node.IndexName,
		EstRows:      est,
		FullScan:     fullScan,
	})
	for _, child := range node.Nodes {
		result.Plan[self].Children = append(result.Plan[self].Children, collectChPlanNodes(child, result, seenIndex))
	}
	return self
}
