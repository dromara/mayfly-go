package dbi

import (
	"context"
	"errors"
	"fmt"
)

// PlanNode 执行计划里的一个节点（各方言字段名不同，这里只保留跨方言都有意义的部分）。
type PlanNode struct {
	// NodeKind 节点类型原文（如 mysql 的 type=ALL、postgres 的 "Seq Scan"），不做归一化猜测
	NodeKind string `json:"nodeKind"`
	// Table 该节点访问的表（取不到为空）
	Table string `json:"table"`
	// AccessMethod 访问方式原文（mysql 的 type/Extra、postgres 的 Node Type）
	AccessMethod string `json:"accessMethod"`
	// IndexName 命中的索引名；空表示未用索引
	IndexName string `json:"indexName"`
	// EstRows 该节点的行数估算（方言未给出时为 0）
	EstRows float64 `json:"estRows"`
	// FullScan 该节点自身是否全表/全索引扫描。判据只能由方言给出（mysql 的 type=ALL/index、
	// pg 的 Seq Scan），前端不得按 NodeKind 文本再猜一遍：那会把方言判定复制成第二套真源，
	// 而它只能对已知方言正确
	FullScan bool `json:"fullScan"`
	// Children 子节点在 Plan 数组里的下标（前序展开，父一定早于子）。
	//
	// 区分两种情形：整个计划里一个 Children 都没有，意味着**该方言没给出父子关系**（只能平铺），
	// 而不是「所有节点都是叶子」；叶子节点本就是空。前端据此选平铺表格还是树，
	// 把无父子信息的情形画成一棵单链假 DAG 是在给用户造事实
	Children []int `json:"children,omitempty"`
}

// ExplainResult 一次执行计划分析的结论。
//
// 关键约定：Rows 是**优化器估算值**而非真实影响行数，EstimateKind 必须如实带出口径，
// 上层展示与策略命中原因都要标注它。把估算当确定值给用户，会让人以为「预计 3 行」就是 3 行，
// 而估算失真（统计信息陈旧、无索引列的等值条件）时差几个数量级是常态。
type ExplainResult struct {
	// Rows 估算的影响/扫描行数（顶层节点）
	Rows float64 `json:"rows"`
	// EstimateKind 估算口径：explain（纯估算，不执行）| explain-analyze（真实执行统计）| unavailable
	EstimateKind string `json:"estimateKind"`
	// FullScan 是否存在全表/全索引扫描（判据由方言给出：mysql type=ALL、postgres Seq Scan 等）
	FullScan bool `json:"fullScan"`
	// UsedIndexes 计划实际用到的索引名（去重）
	UsedIndexes []string `json:"usedIndexes"`
	// Plan 计划节点列表
	Plan []PlanNode `json:"plan,omitempty"`
	// Warnings 方言侧的告警信息（如统计信息缺失、无法分析）
	Warnings []string `json:"warnings,omitempty"`
}

// 估算口径取值：由实现方如实填写，上层展示与策略命中原因都必须带上它。
//
// 只有 explain 这一种是安全的：explain-analyze 会真实执行语句，本模块的预检与策略判定绝不允许用它，
// 定义出来是给未来的只读分析场景（如 SELECT 调优）留口径，而不是给写语句开绿灯
const (
	// EstimateKindExplain 纯计划估算，未执行语句
	EstimateKindExplain = "explain"
)

// ErrExplainUnsupported 该方言未提供执行计划分析能力。
//
// 与 ErrUnsupportedKind 分开：这里的语义是「分析不出来」，上层必须据此标注「无法估算」，
// 不能把 Rows=0 当成「影响 0 行」这个好消息用
var ErrExplainUnsupported = errors.New("explain analysis is not supported for this dialect")

// ExplainAnalyzer 可选能力契约：由方言实现，对一条语句取执行计划与行数估算。
//
// 实现必须只用 EXPLAIN 的**不执行**形态（EXPLAIN / EXPLAIN (FORMAT JSON)），
// 绝不能为了拿实际行数用 EXPLAIN ANALYZE——那会真的把 UPDATE/DELETE 跑一遍。
type ExplainAnalyzer interface {
	// Analyze 分析 sql 的执行计划。sql 由上层保证是已切割出的单条语句
	Analyze(ctx context.Context, sql string) (*ExplainResult, error)
}

// Explain 代理方言的 ExplainAnalyzer 能力；方言未实现则返回 ErrExplainUnsupported。
//
// 与 ListSessions 同一手法（运行期类型断言，不虚构空结果）：返回 err 时上层不得把零值
// ExplainResult 当成「估算 0 行」使用
func (m *MetadataReader) Explain(ctx context.Context, sql string) (*ExplainResult, error) {
	analyzer, ok := m.provider.(ExplainAnalyzer)
	if !ok {
		return nil, fmt.Errorf("%w: provider %T", ErrExplainUnsupported, m.provider)
	}
	res, err := analyzer.Analyze(ctx, sql)
	if err != nil {
		return nil, err
	}
	if res.EstimateKind == "" {
		return nil, fmt.Errorf("%w: provider %T returned no estimate kind", ErrExplainUnsupported, m.provider)
	}
	return res, nil
}
