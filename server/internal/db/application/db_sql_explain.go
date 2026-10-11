package application

// SQL 执行计划取数（编辑器/工单「查看执行计划」入口）。
//
// 计划是「每条语句一份大结构」，单独一个只读入口按需在用户查看时才逐条 EXPLAIN，
// 不给执行路径压上计划开销。与执行期共用同一套语句切割与同一份分类实现，
// 保证看到的第 N 条与执行时的第 N 条是同一条。
//
// 只读边界：EXPLAIN 一律不带 ANALYZE（写死在各方言实现里），看一条 DELETE 的计划不会真删掉数据，
// 这也是本入口可以复用「能执行SQL脚本」这一档权限的前提。

import (
	"context"

	"mayfly-go/internal/db/application/sqlutil"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/i18n"
)

// ExplainPlan 单条语句的执行计划取数结果。
//
// Result 与 Note 恰好一个非空：拿不到计划时必须带上 Note，前端不能把「没有计划」渲染成
// 「这条语句没有扫描」——那是把观测缺口说成好消息
type ExplainPlan struct {
	SQL      string `json:"sql"`
	StmtType string `json:"stmtType"`

	// Result 方言给出的计划：Rows 是优化器估算值，口径见其 EstimateKind
	Result *dbi.ExplainResult `json:"result,omitempty"`
	// Note 本次拿不到计划的原因说明（方言不支持、语句本身不可分析、库侧报错）
	Note string `json:"note,omitempty"`
}

// Explain 逐条取语句的执行计划，单条取不到不影响其余条。
//
// 逐条容错而不是整批报错：一个脚本里混着一条该方言 EXPLAIN 不支持的语句（如 SET/SHOW）时，
// 其余语句的计划对用户仍然有用，整批失败等于把可用的一半也丢掉
func (d *dbAppImpl) Explain(ctx context.Context, dbId uint64, dbName string, sqls []string) ([]*ExplainPlan, error) {
	conn, err := d.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, err
	}
	statements, err := sqlutil.SplitConnStatements(ctx, conn, sqls)
	if err != nil {
		return nil, err
	}
	dialect := conn.GetDialect()

	plans := make([]*ExplainPlan, 0, len(statements))
	for _, sql := range statements {
		_, stmtType := sqlutil.ClassifyStatement(dialect, sql)
		plan := &ExplainPlan{SQL: sql, StmtType: stmtType}
		res, explainErr := conn.Metadata().Explain(ctx, sql)
		if explainErr != nil {
			plan.Note = i18n.TC(ctx, imsg.ErrExplainUnavailable, "reason", sqlutil.BriefReason(explainErr))
		} else {
			plan.Result = res
		}
		plans = append(plans, plan)
	}
	return plans, nil
}
