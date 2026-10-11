package oracle

// oracle 执行计划分析（可选能力 ExplainAnalyzer 的 oracle 实现）。
//
// 用 EXPLAIN PLAN FOR <stmt>：只编译生成计划、**不执行**语句（对 UPDATE/DELETE 安全），计划写入 PLAN_TABLE；
// 再用随机 statement_id 隔离本次并发写入并回读，结束即删除本次行避免污染共享 PLAN_TABLE。
//
// 全表扫描判据：operation='TABLE ACCESS' 且 options='FULL'。INDEX 类算子的 object_name 是索引名（记入
// UsedIndexes），TABLE ACCESS 的 object_name 才是表名——二者不能混填。Rows 取根节点（id=0）的 cardinality
// （优化器估算），故 EstimateKind 固定 explain。

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

var _ dbi.ExplainAnalyzer = (*OracleMetadata)(nil)

// oraclePlanRow PLAN_TABLE 一行（一次语句的某个计划步骤）。
type oraclePlanRow struct {
	id, parent  int
	operation   string
	options     string
	objectName  string
	cardinality float64
}

func (od *OracleMetadata) Analyze(ctx context.Context, sql string) (*dbi.ExplainResult, error) {
	stmt := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(sql), ";"))
	if stmt == "" {
		return nil, errorx.NewBiz("sql to analyze is empty")
	}
	upper := strings.ToUpper(stmt)
	if strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "DESC") {
		return nil, errorx.NewBiz("the statement to analyze must not be an EXPLAIN/DESC itself")
	}

	stmtId, err := oraclePlanStatementId()
	if err != nil {
		return nil, err
	}

	// 生成计划（不执行）
	if _, err := od.di.ExecContext(ctx, "EXPLAIN PLAN SET STATEMENT_ID = '"+stmtId+"' FOR "+stmt); err != nil {
		return nil, err
	}
	// 无论回读成败都清理本次 statement_id 行，避免污染共享 PLAN_TABLE；用独立 ctx 保证收尾执行
	defer func() {
		_, _ = od.di.ExecContext(context.Background(), "DELETE FROM plan_table WHERE statement_id = '"+stmtId+"'")
	}()

	_, res, err := od.di.QueryContext(ctx, `SELECT p.id AS "id", p.parent_id AS "parent",
		p.operation AS "operation", p.options AS "options",
		COALESCE(p.object_name, '') AS "objectName",
		COALESCE(p.cardinality, 0) AS "cardinality"
		FROM plan_table p WHERE p.statement_id = '`+stmtId+`' ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errorx.NewBiz("explain returned no plan rows")
	}

	rows := make([]oraclePlanRow, 0, len(res))
	for _, r := range res {
		rows = append(rows, oraclePlanRow{
			id:          cast.ToInt(r["id"]),
			parent:      cast.ToInt(r["parent"]),
			operation:   cast.ToString(r["operation"]),
			options:     cast.ToString(r["options"]),
			objectName:  cast.ToString(r["objectName"]),
			cardinality: cast.ToFloat64(r["cardinality"]),
		})
	}
	return buildOracleExplainResult(rows), nil
}

// buildOracleExplainResult 依 id/parent_id 还原父子边（plan_table 按 id 前序，父 id 恒小于子 id）。
func buildOracleExplainResult(rows []oraclePlanRow) *dbi.ExplainResult {
	result := &dbi.ExplainResult{EstimateKind: dbi.EstimateKindExplain}
	idToIdx := make(map[int]int, len(rows))
	seenIndex := make(map[string]bool)

	for i, r := range rows {
		fullScan := strings.EqualFold(r.operation, "TABLE ACCESS") && strings.EqualFold(r.options, "FULL")
		if fullScan {
			result.FullScan = true
		}
		isIndex := strings.HasPrefix(strings.ToUpper(r.operation), "INDEX ")
		table, index := r.objectName, ""
		if isIndex {
			table, index = "", r.objectName
			if index != "" && !seenIndex[index] {
				seenIndex[index] = true
				result.UsedIndexes = append(result.UsedIndexes, index)
			}
		}
		if r.id == 0 {
			result.Rows = r.cardinality
		}
		result.Plan = append(result.Plan, dbi.PlanNode{
			NodeKind:     r.operation,
			Table:        table,
			AccessMethod: strings.TrimSpace(r.operation + " " + r.options),
			IndexName:    index,
			EstRows:      r.cardinality,
			FullScan:     fullScan,
		})
		idToIdx[r.id] = i
	}

	for i, r := range rows {
		if r.parent == r.id {
			continue
		}
		if pi, ok := idToIdx[r.parent]; ok && pi != i {
			result.Plan[pi].Children = append(result.Plan[pi].Children, i)
		}
	}
	return result
}

// oraclePlanStatementId 生成随机、仅含安全字符的 statement_id（隔离并发 EXPLAIN，且可安全拼入语句）。
//
// 总长必须不超过 30 字节：真库实测 31 字节会让整条 EXPLAIN PLAN 直接报 ORA-00972
// （identifier exceeds the maximum length of 30 bytes），表现为 Oracle 上「查看执行计划」每次都失败
func oraclePlanStatementId() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "mayfly_" + hex.EncodeToString(b[:]), nil
}
