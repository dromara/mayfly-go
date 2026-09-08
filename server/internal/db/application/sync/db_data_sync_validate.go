package sync

import (
	"context"
	"regexp"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/errorx"
)

// getSrcDialect 获取同步任务源库方言（dbm.Conn对相同连接信息复用连接池，成本可控）
func (app *DataSyncAppImpl) getSrcDialect(task *entity.DataSyncTask) (dbi.Dialect, error) {
	srcConn, err := app.dbApp.GetDbConn(context.Background(), uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		return nil, errorx.NewBizf("failed to connect to the source database: %s", err.Error())
	}
	return srcConn.GetDialect(), nil
}

// identifierReg 同步增量字段标识符白名单：形如 id / a.id。
// UpdField/UpdFieldSrc会直接拼入where与order by子句，白名单杜绝SQL注入面
var identifierReg = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// validateDataSyncSql 校验同步任务的DataSql与增量字段合法性（保存与运行双重拦截）：
//  1. UpdField/UpdFieldSrc必须匹配标识符白名单
//  2. DataSql必须是**单条**SELECT/WITH查询（拒绝DML/DDL/多语句）
func validateDataSyncSql(dialect dbi.Dialect, task *entity.DataSyncTask) error {
	if task.UpdField != "" && !identifierReg.MatchString(task.UpdField) {
		return errorx.NewBizf("invalid updField [%s]: only identifiers like [id] or [a.id] are allowed", task.UpdField)
	}
	if task.UpdFieldSrc != "" && !identifierReg.MatchString(task.UpdFieldSrc) {
		return errorx.NewBizf("invalid updFieldSrc [%s]: only identifiers like [id] or [a.id] are allowed", task.UpdFieldSrc)
	}

	if strings.TrimSpace(task.DataSql) == "" {
		return errorx.NewBiz("data sql is required")
	}

	// 先按方言切割器切分，保证仅一条语句（解析器只解析单条，多条语句的尾部可能被忽略）
	stmtCount := 0
	if err := dialect.GetSQLSplitter().SplitSQL(strings.NewReader(task.DataSql), func(s string) error {
		if strings.TrimSpace(s) != "" {
			stmtCount++
		}
		return nil
	}); err != nil {
		return errorx.NewBizf("data sql split failed: %s", err.Error())
	}
	if stmtCount != 1 {
		return errorx.NewBizf("data sql must be a single statement, got %d statements", stmtCount)
	}

	stmt, err := dialect.GetSQLParser().Parse(task.DataSql)
	if err != nil {
		return errorx.NewBizf("data sql parse failed: %s", err.Error())
	}
	if stmt == nil {
		// 防御：解析器异常场景可能返回nil stmt，直接解引用会panic
		return errorx.NewBiz("data sql parse failed: empty statement")
	}
	switch stmt.StmtKind() {
	case sqlstmt.KindSelect:
		sel, ok := stmt.(*sqlstmt.SelectStmt)
		if !ok {
			return nil
		}
		return validateDataSyncAppendable(sel, task)
	case sqlstmt.KindWith:
		// WITH语句解析结果无法内省顶层SELECT形态，而Run时是在语句尾部拼接条件，
		// 无法可靠判定顶层WHERE与尾部子句位置，只能保守拦截会产生非法SQL的组合：
		//   - 配置了增量字段：追加"and/or order by"依赖顶层WHERE存在与否，无法判定，直接拒绝
		//   - 未配置增量字段但全文无where且有尾部子句：会追加"where 1=1"落在尾部子句之后，拒绝
		if task.UpdField != "" {
			return errorx.NewBizf("data sql with with-clause does not support incremental updField: the runner appends conditions at the statement tail which cannot be reliably positioned")
		}
		if !whereReg.MatchString(task.DataSql) && dataSyncTrailingClauseReg.MatchString(task.DataSql) {
			return errorx.NewBizf("data sql with with-clause cannot contain group by/having/order by/limit clauses when it has no where: appended where would produce invalid sql")
		}
		return nil
	default:
		return errorx.NewBizf("data sql must be a single select query, got [%s]", stmt.StmtKind())
	}
}

// dataSyncTrailingClauseReg WITH语句尾部子句关键字探测（词边界匹配，无法内省时的保守兜底）
var dataSyncTrailingClauseReg = regexp.MustCompile(`(?i)\b(group\s+by|having|order\s+by|limit|offset)\b`)

// validateDataSyncAppendable 校验Run时尾部拼接不会产生非法SQL。
//
// 同步执行按 `dataSql [where 1=1] [and upd > val] [order by upd asc]` 形态在语句尾部追加条件：
//   - 无WHERE时补"where 1=1"
//   - 配置了增量字段时追加"and upd > val"与"order by upd asc"
//
// 语句顶层存在GROUP BY/HAVING/ORDER BY/LIMIT时，追加内容必然落在这些子句之后（非法SQL）；
// 子查询内的同名子句不影响（仅检查顶层）。已含WHERE且未配置增量字段时不追加任何内容，无需限制。
// 注意：无WHERE且有GROUP BY时即使未配置增量字段也需拒绝（"where 1=1"同样拼在group by之后）
func validateDataSyncAppendable(sel *sqlstmt.SelectStmt, task *entity.DataSyncTask) error {
	appends := sel.Where == nil || task.UpdField != ""
	if !appends {
		return nil
	}
	if len(sel.GroupBy) > 0 || sel.Having != nil || len(sel.OrderBy) > 0 || sel.Limit != nil {
		return errorx.NewBizf("data sql cannot contain top-level group by/having/order by/limit clauses when it has no where or has incremental updField: the sync runner appends conditions after the statement")
	}
	return nil
}

// dataSqlHasWhere 判断DataSql是否已含WHERE条件（用于决定是否补"where 1=1"）。
//
// 优先用源方言解析器判定（SelectStmt.Where非空），替代旧(?i)where正则——
// 旧正则会把不含where条件但字段名/表名/字符串字面量含"where"子串的SQL误判为已有条件，
// 导致后续拼接"and updField > x"时产生非法SQL。
// WITH语句（解析结果无法内省where）与解析失败场景降级正则兜底
func dataSqlHasWhere(dialect dbi.Dialect, task *entity.DataSyncTask) bool {
	if dialect != nil {
		if stmt, err := dialect.GetSQLParser().Parse(task.DataSql); err == nil {
			if sel, ok := stmt.(*sqlstmt.SelectStmt); ok {
				return sel.Where != nil
			}
			return whereReg.MatchString(task.DataSql)
		}
	}
	return whereReg.MatchString(task.DataSql)
}
