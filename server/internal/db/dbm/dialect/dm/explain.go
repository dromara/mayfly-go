package dm

// dm（达梦）执行计划分析（可选能力 ExplainAnalyzer 的 dm 实现）。
//
// 用 EXPLAIN <stmt>：达梦的 EXPLAIN 只分析不执行语句，返回文本形态计划（每行一个步骤）。
// 文本计划不含每步行数估算，故 Rows 置 0 并如实记一条 warning——绝不把 0 当成「估算 0 行」的好消息。
// 全表扫描判据：计划文本出现 FULL SCAN / CLUSTER SCAN（达梦对无索引/全表访问的算子命名）。
//
// ⚠️ 验证状态：本环境无达梦实例，EXPLAIN 输出列名与算子文本按 DM8 文档编写，接入真实 DM8 前需核对。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

var _ dbi.ExplainAnalyzer = (*DMMetadata)(nil)

func (dd *DMMetadata) Analyze(ctx context.Context, sql string) (*dbi.ExplainResult, error) {
	stmt := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(sql), ";"))
	if stmt == "" {
		return nil, errorx.NewBiz("sql to analyze is empty")
	}
	upper := strings.ToUpper(stmt)
	if strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "DESC") {
		return nil, errorx.NewBiz("the statement to analyze must not be an EXPLAIN/DESC itself")
	}

	_, res, err := dd.di.QueryContext(ctx, "EXPLAIN "+stmt)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errorx.NewBiz("explain returned no plan")
	}

	result := &dbi.ExplainResult{EstimateKind: dbi.EstimateKindExplain}
	for _, row := range res {
		// 取该行第一个非空文本作为计划步骤（EXPLAIN 输出列名随版本不同，按值而非列名取更稳）
		line := firstTextValue(row)
		if line == "" {
			continue
		}
		fullScan := dmLineIsFullScan(line)
		if fullScan {
			result.FullScan = true
		}
		result.Plan = append(result.Plan, dbi.PlanNode{
			NodeKind:     line,
			AccessMethod: line,
			FullScan:     fullScan,
		})
	}
	if len(result.Plan) == 0 {
		return nil, errorx.NewBiz("explain returned no plan step")
	}
	// 文本计划无行数估算：如实标注，避免上层把 Rows=0 读成「影响 0 行」
	result.Warnings = append(result.Warnings, "dm EXPLAIN text plan provides no row-count estimate")
	return result, nil
}

// dmLineIsFullScan 计划行含全表/聚簇扫描算子名即判全表扫描。
func dmLineIsFullScan(line string) bool {
	up := strings.ToUpper(line)
	return strings.Contains(up, "FULL SCAN") || strings.Contains(up, "CLUSTER SCAN") || strings.Contains(up, "TABLE SCAN")
}

// firstTextValue 返回一行结果里第一个非空文本字段值。
func firstTextValue(row map[string]any) string {
	for _, v := range row {
		if s := strings.TrimSpace(cast.ToString(v)); s != "" {
			return s
		}
	}
	return ""
}
