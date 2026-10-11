package sqlexec

// SQL 触发策略「需查库才能算出的事实」与检查项求值。
//
// 设计要点（执行期判定与测试求值共用一份实现，是本文件存在的核心理由）：
//   - 估算行数 / 是否全表扫描 / 目标表有无主键 / 目标表规模这些**要查库**的事实，
//     一律注册成触发引擎的惰性派生字段（见 dbTriggerFields 的 Resolve），执行期策略判定按同一份取值。
//   - reviewStatement/reviewWithChecks 用同一批 CheckDef 与默认参数对单条语句求值，
//     是这些检查项行为的验证入口（单测/集成测试据此钉死 fail-closed、无主键点名、DDL 走表统计等口径）。
//   - 派生字段的惰性意味着成本可控：没配相应策略的库不会为一条语句多跑一次 EXPLAIN。

import (
	"context"

	"mayfly-go/internal/db/application/sqlutil"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/imsg"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"

	"github.com/spf13/cast"
)

// 需要查库才能得出的派生字段 key
const (
	// dbFieldEstimatedRows 优化器估算的行数（口径见 estimateKind）
	dbFieldEstimatedRows = "estimatedRows"
	// dbFieldFullScan 计划里是否存在全表/全索引扫描
	dbFieldFullScan = "fullScan"
	// dbFieldTargetTableNoPk 目标表是否缺主键
	dbFieldTargetTableNoPk = "targetTableNoPk"
	// dbFieldTargetTableRows 目标表规模估算行数（各目标表最大值）
	dbFieldTargetTableRows = "targetTableRows"
	// dbFieldEstimateKind 行数估算口径：告诉用户这是未执行的优化器估算，不是真实影响行数
	dbFieldEstimateKind = "estimateKind"
)

// 求值上下文里携带的库定位信息。
// 都是可安全序列化进日志的标量，不放连接对象：连接对象一旦进入事实集合，
// 日志/回显里的 JSON 序列化要么报错要么吐出巨型结构
const (
	dbAttrDbId   = "dbId"
	dbAttrDbName = "dbName"
	// dbAttrExplainCached 计划结果的缓存位（下划线前缀表示内部位，不作为可条件化的字段）。
	// 同一语句的 estimatedRows 与 fullScan 两个派生字段共用它，避免重复 EXPLAIN
	dbAttrExplainCached = "_explainResult"
)

// ReviewFinding 一条预检命中结果。
//
// Severity 取检查项自带的建议默认级别；最终处置仍由该库绑定的策略决定，
// 预检面板不假装自己有裁决权
type ReviewFinding struct {
	SQL      string `json:"sql"`
	StmtType string `json:"stmtType"`
	CheckKey string `json:"checkKey,omitempty"`
	TitleKey string `json:"titleKey,omitempty"`
	Reason   string `json:"reason,omitempty"`
	// Severity 检查项的建议默认级别，数值与流程侧严重度枚举同源，由前端映射文案
	Severity flowentity.Severity `json:"severity,omitempty"`
	Detail   map[string]any      `json:"detail,omitempty"`
	// Note 无法判定这类「本次没给出结论」的说明，与命中分开：把「算不出来」显示成「未命中」
	// 是最危险的假安心
	Note string `json:"note,omitempty"`
}

// reviewStatement 用该场景已注册的全部检查项预检单条语句
func reviewStatement(ctx context.Context, dialect dbi.Dialect, dbId uint64, dbName, sql string) ([]*ReviewFinding, error) {
	return reviewWithChecks(ctx, dialect, dbId, dbName, sql, trigger.ChecksOf(DbSQLExecFlowBizType))
}

// reviewWithChecks 用检查项默认参数评估给定的检查项集合。
//
// 只收 dialect 而不是 conn：解析/切割确实需要方言，但「要查哪个库」已经写在事实里（dbId + 库名），
// 由派生字段惰性去取连接。把 conn 传进来会让这条链路在没库可连时根本进不来，
// 而预检必须能回答「查不到就只能判定到这一步」而不是一起报错。
//
// 检查项集合由参数给出而不是就地取注册表：注册表禁止重复注册，测试无法注入一个会报错的检查项
// 来驱动下面的「无法判定」分支，而那条分支正是「不把观测缺口显示成安全」的唯一保障
func reviewWithChecks(ctx context.Context, dialect dbi.Dialect, dbId uint64, dbName, sql string, checks []trigger.CheckDef) ([]*ReviewFinding, error) {
	stmt, stmtType := sqlutil.ClassifyStatement(dialect, sql)

	tc := trigger.NewContext(ctx, DbSQLExecFlowBizType, 0, map[string]string{
		dbFieldStmtType: stmtType,
		dbFieldSQL:      sql,
	}, contextx.GetLoginAccount(ctx)).WithAttributes(reviewFacts(dbId, dbName, sql, stmt))

	result := make([]*ReviewFinding, 0, 2)
	for _, check := range checks {
		hit, err := check.Evaluate(ctx, tc, defaultCheckParams(check))
		if err != nil {
			// 无法判定如实回报：不能把它折叠成「不命中」
			logx.WarnfContext(ctx, "sql review: check [%s] could not be evaluated: %s", check.Key, err.Error())
			result = append(result, &ReviewFinding{
				SQL:      sql,
				StmtType: stmtType,
				CheckKey: check.Key,
				TitleKey: check.TitleKey,
				Note:     i18n.TC(ctx, imsg.ErrReviewCheckNotEvaluated, "reason", sqlutil.BriefReason(err)),
			})
			continue
		}
		if !hit {
			continue
		}
		result = append(result, &ReviewFinding{
			SQL:      sql,
			StmtType: stmtType,
			CheckKey: check.Key,
			TitleKey: check.TitleKey,
			Reason:   i18n.TC(ctx, check.Summary),
			Severity: check.Default,
			Detail:   reviewDetail(ctx, tc, check.Key),
		})
	}
	return result, nil
}

// reviewFacts 与执行期同一份事实构造（sqlTriggerFacts + 库定位），保证两边口径不漂移
func reviewFacts(dbId uint64, dbName, sql string, stmt sqlstmt.Stmt) map[string]any {
	attributes := sqlTriggerFacts(sql, stmt)
	attributes[dbAttrDbId] = dbId
	attributes[dbAttrDbName] = dbName
	return attributes
}

// defaultCheckParams 检查项默认参数（与策略页未填参数时的取值同源）
func defaultCheckParams(check trigger.CheckDef) map[string]any {
	params := make(map[string]any, len(check.Params))
	for _, param := range check.Params {
		if param.Default != nil {
			params[param.Key] = param.Default
		}
	}
	return params
}

// reviewDetail 命中详情：带上已算出的派生值，让界面能说明「为什么命中」而不只报一句规则名。
//
// 行数/计划这一栏只对按行数判定的检查项有意义：DDL 命中「大表结构变更」时附一句「拿不到行数
// 估算」是无关噪声，还会让人以为 DDL 本应能估算行数。取不到估算时必须明说「取不到」，
// 详情里缺行数与「估出 0 行」在用户眼里长得一样，后者是假安心
func reviewDetail(ctx context.Context, tc *trigger.Context, checkKey string) map[string]any {
	detail := map[string]any{}
	if checkKey == dbCheckRowsExceeds || checkKey == dbCheckFullScan {
		if res, err := explainOnce(tc); err == nil {
			detail["estimatedRows"] = res.Rows
			detail["estimateKind"] = res.EstimateKind
			detail["fullScan"] = res.FullScan
			if len(res.UsedIndexes) > 0 {
				detail["usedIndexes"] = res.UsedIndexes
			}
		} else {
			// 详情里缺行数字段不等于「估算为 0 行」，必须把「取不到」本身说出来
			detail["estimateNote"] = i18n.TC(ctx, imsg.ReviewEstimateUnavailable, "reason", sqlutil.BriefReason(err))
		}
	}
	if tables, err := noPrimaryKeyTables(ctx, tc); err == nil && len(tables) > 0 {
		detail["noPkTables"] = tables
	}
	return detail
}

// explainOnce 取该语句的执行计划，同一求值上下文只分析一次。
//
// 失败原样上抛（含方言不支持）：调用方必须区分「算出来是 0 行」与「算不出来」，
// 把算不出来当 0 行会让「影响行数超阈值需审批」这类规则在不支持的方言上静默永不命中
func explainOnce(tc *trigger.Context) (*dbi.ExplainResult, error) {
	if cached, ok := tc.Attribute(dbAttrExplainCached); ok {
		if res, ok := cached.(*dbi.ExplainResult); ok {
			return res, nil
		}
	}

	sql, _ := tc.RawValue(dbFieldSQL)
	if stringx.TrimSpaceAndBr(sql) == "" {
		return nil, errorx.NewBizI(tc.GoContext(), imsg.ErrReviewNoSqlToAnalyze)
	}

	conn, err := connFromFacts(tc.GoContext(), tc)
	if err != nil {
		return nil, err
	}
	res, err := conn.Metadata().Explain(tc.GoContext(), sql)
	if err != nil {
		return nil, err
	}
	tc.Attributes[dbAttrExplainCached] = res
	return res, nil
}

// connFromFacts 由事实里的库定位取连接：拿不到即「无法判定」，不做放行假设
func connFromFacts(ctx context.Context, tc *trigger.Context) (*dbi.DbConn, error) {
	rawDbId, ok := tc.Attribute(dbAttrDbId)
	if !ok {
		return nil, errorx.NewBizI(ctx, imsg.ErrReviewNoTargetDb)
	}
	dbId := cast.ToUint64(rawDbId)
	if dbId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrReviewNoTargetDb)
	}
	rawDbName, _ := tc.Attribute(dbAttrDbName)
	return getDbConnForFacts(ctx, dbId, cast.ToString(rawDbName))
}

// getDbConnForFacts 派生字段取库连接的入口。
//
// 单独成变量只为让集成测试能在不启动 IOC 容器的情况下钉住「预检与执行同口径」这条契约；
// 生产路径始终走 IOC 解析出的 Db 应用，测试替换不影响线上行为
var getDbConnForFacts = func(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error) {
	return ioc.Get[DbConnReader]().GetDbConn(ctx, dbId, dbName)
}

// targetTablesOf 取本次求值已解析出的目标表清单
func targetTablesOf(tc *trigger.Context) []string {
	tables, ok := tc.Attribute(dbFieldTargetTables)
	if !ok {
		return nil
	}
	list, _ := tables.([]string)
	return list
}

// noPrimaryKeyTables 返回目标表里缺主键的表。
// 内省失败的表跳过而非断言「无主键」：把查不到当无主键会让该检查在权限受限的库上恒命中
func noPrimaryKeyTables(ctx context.Context, tc *trigger.Context) ([]string, error) {
	tables := targetTablesOf(tc)
	if len(tables) == 0 {
		return nil, nil
	}
	conn, err := connFromFacts(ctx, tc)
	if err != nil {
		return nil, err
	}
	missing := make([]string, 0, len(tables))
	for _, table := range tables {
		pks, pkErr := conn.Metadata().GetPrimaryKeys(table)
		if pkErr != nil {
			logx.WarnfContext(ctx, "db trigger facts: failed to read primary keys of [%s]: %s", table, pkErr.Error())
			continue
		}
		if len(pks) == 0 {
			missing = append(missing, table)
		}
	}
	return missing, nil
}

// maxTargetTableRows 目标表规模估算的最大值；一个都取不到时返回错误，不用 0 冒充「空表」
func maxTargetTableRows(ctx context.Context, tc *trigger.Context) (int, error) {
	tables := targetTablesOf(tc)
	if len(tables) == 0 {
		return 0, errorx.NewBizI(ctx, imsg.ErrReviewNoTargetTable)
	}
	conn, err := connFromFacts(ctx, tc)
	if err != nil {
		return 0, err
	}
	maxRows := -1
	for _, table := range tables {
		got, gotErr := conn.Metadata().GetTables(table)
		if gotErr != nil || len(got) == 0 {
			logx.WarnfContext(ctx, "db trigger facts: failed to read table statistics of [%s]: %v", table, gotErr)
			continue
		}
		// TableRows 本身可能是方言侧估算值，这里只用于「大表 DDL 需额外确认」的量级判断
		if got[0].TableRows > maxRows {
			maxRows = got[0].TableRows
		}
	}
	if maxRows < 0 {
		return 0, errorx.NewBizI(ctx, imsg.ErrReviewNoTableStats)
	}
	return maxRows, nil
}
