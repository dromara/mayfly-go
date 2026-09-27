package sync

import (
	"context"
	"regexp"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

// getSrcConn 获取同步任务源库连接（dbm.Conn 对相同连接信息复用连接池，成本可控），
// 供 Save 双重拦截时的方言解析与元数据探测（如索引存在性）共用。
func (app *DataSyncAppImpl) getSrcConn(task *entity.DataSyncTask) (*dbi.DbConn, error) {
	srcConn, err := app.dbApp.GetDbConn(context.Background(), uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		return nil, errorx.NewBizf("failed to connect to the source database: %s", err.Error())
	}
	return srcConn, nil
}

// singleTableForIndex 从解析后的顶层 SELECT 提取可目标准确定位的单表引用。
// 返回 ok=false 的三种情况，均无法确定 UpdField 属于哪张物理表：
//   - FROM 非单一元素（子查询入口、多表逗号列举）
//   - 含 JOIN 子句（UpdField 可能属于任一侧，无法固定）
//   - 顶层 Name 为空（子查询作为 FROM 主体，无表名可给 MetadataReader）
//
// UNION 亦会因 len(sel.Unions)>0 拒绝，避免只校验左侧、右侧无索引仍扫全表。
func singleTableForIndex(sel *sqlstmt.SelectStmt) (schema, name string, ok bool) {
	if sel == nil || len(sel.From) != 1 || len(sel.Joins) > 0 || len(sel.Unions) > 0 {
		return "", "", false
	}
	ref := sel.From[0]
	// 解析器把 `FROM (SELECT ...) t` 整体塞进 Name（含括号与 alias），
	// 用与增量字段一致的标识符白名单二次拒绝非物理表名（子查询、函数、表达式等）。
	if !plainIdentReg.MatchString(ref.Name) {
		return "", "", false
	}
	if ref.Schema != "" && !plainIdentReg.MatchString(ref.Schema) {
		return "", "", false
	}
	return ref.Schema, ref.Name, true
}

// plainIdentReg 单个未加引号的 SQL 标识符（表名/schema 名）：字母或下划线开头，后接字母数字下划线。
// 与 identifierReg（用于列，允许 `a.b` 两级）不同——这里只匹配单段。
var plainIdentReg = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateIncrementalFieldIndex 校验增量字段在源表上至少作为某条索引的首列存在（P1）。
//
// 目的：从根上堵“每轮全表扫 + filesort”——buildSyncQuery 恒拼 `WHERE UpdField 边界 水位 ORDER BY UpdField`，
// 无索引时 MySQL 走全表扫 + filesort，高频 cron 下会稳定拉高源库负载。保存时拦住比运行时才发现代价低。
//
// 启发式而非完备判据：
//   - UpdField 为空（无水位，如首轮/全量模式）→ 跳过；
//   - SkipIndexValidation=true → 跳过（视图/函数索引/特殊部署下的逃生阀）；
//   - DataSQL 不是单表无 JOIN 无 UNION 的顶层 SELECT → 跳过（无法确定 UpdField 归属）；
//   - 元数据查询失败（权限/临时不可达）→ 降级为 Warn，不阻断保存（避免非确定错误把用户拒在门外）。
//
// 主键列同样满足（最左前缀可走索引扫描）：mysql 的 GetTableIndex 含 PRIMARY 行可天然命中；
// pg/mssql/oracle/dm/sqlite 的 GetTableIndex 有意排除主键索引（口径与 dump/索引展示一致），
// 故索引未命中时还需调 GetPrimaryKeys 兑底判定首列主键，否则仅含主键的表在非 mysql 源上会被误拒。
// 导出供黑盒集成测试驱动（与 Save 链路同一实现）。
func ValidateIncrementalFieldIndex(ctx context.Context, srcConn *dbi.DbConn, task *entity.DataSyncTask) error {
	if task.UpdField == "" || task.GetSkipIndexValidation() {
		return nil
	}
	stmt, err := srcConn.GetDialect().GetSQLParser().Parse(task.DataSQL)
	if err != nil {
		// 解析失败交给上层的 validateDataSyncSQL 处理，本函数不重复报错
		return nil
	}
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	if !ok {
		return nil
	}
	schema, table, ok := singleTableForIndex(sel)
	if !ok {
		return nil
	}
	// 列名可能带 `a.id` 形式的表限定前缀，白名单已限到两级，取尾列为物理列名
	physicalCol := task.UpdField
	if i := strings.LastIndexByte(physicalCol, '.'); i >= 0 {
		physicalCol = physicalCol[i+1:]
	}
	indexes, err := srcConn.Metadata().GetTableIndex(table)
	if err != nil {
		// 元数据不可达（权限/临时故障）降级为 warn，不阻断保存；实际同步失败会在运行时报
		logx.WarnfContext(ctx, "skip incremental field index validation for table %s.%s: %s", schema, table, err.Error())
		return nil
	}
	// 方言 GetTableIndex 已按索引名合并，ColumnName 为逗号连接的列集（见 mysql/postgres metadata）；
	// 首列判断需按逗号切分取首段，不能与合并后的整列串等值比较（否则复合索引 (upd_time, id) 会被误拒）。
	for _, idx := range indexes {
		firstCol := idx.ColumnName
		if comma := strings.IndexByte(firstCol, ','); comma >= 0 {
			firstCol = firstCol[:comma]
		}
		firstCol = strings.TrimSpace(firstCol)
		if strings.EqualFold(firstCol, physicalCol) {
			return nil
		}
	}

	// 兑底：索引列表不含主键的方言（pg/mssql/oracle/dm/sqlite）在此判定首列主键。
	// 复合主键只认首列：非首列的水位谓词无法利用最左前缀扫描，拒绝语义与索引首列一致；
	// GetPrimaryKeys 的列序依赖各白 GetColumns 实现，若次序异常最坏是保守拒绝，不会误放行。
	pks, pkErr := srcConn.Metadata().GetPrimaryKeys(table)
	if pkErr != nil {
		// 与索引探测同语义：主键元数据不可达降级为 warn，不阻断保存
		logx.WarnfContext(ctx, "skip incremental field primary key fallback for table %s.%s: %s", schema, table, pkErr.Error())
		return nil
	}
	if len(pks) > 0 && strings.EqualFold(strings.TrimSpace(pks[0]), physicalCol) {
		return nil
	}
	return errorx.NewBizf("incremental field %q is not the leading column of any index or the leading primary key column on source table %s; every run will scan the full table with filesort. add an index, or enable 'skip index validation' to override this check.", physicalCol, table)
}

// identifierReg 同步增量字段标识符白名单：形如 id / a.id。
// UpdField/UpdFieldSrc会直接拼入where与order by子句，白名单杜绝SQL注入面
var identifierReg = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// validateDataSyncSQL 校验同步任务的DataSQL与增量字段合法性（保存与运行双重拦截）：
//  1. UpdField/UpdFieldSrc必须匹配标识符白名单
//  2. DataSQL必须是**单条**SELECT/WITH查询（拒绝DML/DDL/多语句）
//  3. CursorInclusivity 与同步模式的组合不得语义矛盾（见 validateCursorSemantics）
//  4. SleepBetweenBatchesMs 不得超出上限
func validateDataSyncSQL(dialect dbi.Dialect, task *entity.DataSyncTask) error {
	if err := NewFilterEngine(task.FilterCondition).Validate(); err != nil {
		return errorx.NewBizf("invalid filter condition: %s", err.Error())
	}
	if sleepMs := task.GetSleepBetweenBatchesMs(); sleepMs < 0 || sleepMs > entity.MaxSleepBetweenBatchesMs {
		return errorx.NewBizf("invalid sleepBetweenBatchesMs [%d]: must be within [0, %d]", sleepMs, entity.MaxSleepBetweenBatchesMs)
	}
	if err := validateCursorSemantics(task); err != nil {
		return err
	}
	if task.UpdField != "" && !identifierReg.MatchString(task.UpdField) {
		return errorx.NewBizf("invalid updField [%s]: only identifiers like [id] or [a.id] are allowed", task.UpdField)
	}
	if task.UpdFieldSecondary != "" && !identifierReg.MatchString(task.UpdFieldSecondary) {
		return errorx.NewBizf("invalid updFieldSecondary [%s]: only identifiers like [id] or [a.id] are allowed", task.UpdFieldSecondary)
	}
	if task.UpdFieldSrc != "" && !identifierReg.MatchString(task.UpdFieldSrc) {
		return errorx.NewBizf("invalid updFieldSrc [%s]: only identifiers like [id] or [a.id] are allowed", task.UpdFieldSrc)
	}
	// SoftDeleteField 同样为标识符，需白名单校验（防止 SQL 注入）
	if task.SoftDeleteField != "" && !identifierReg.MatchString(task.SoftDeleteField) {
		return errorx.NewBizf("invalid softDeleteField [%s]: only identifiers like [id] or [a.id] are allowed", task.SoftDeleteField)
	}
	// BiDirTimestampField 用于冲突检测 SELECT，需白名单校验
	if task.BiDirTimestampField != "" && !identifierReg.MatchString(task.BiDirTimestampField) {
		return errorx.NewBizf("invalid biDirTimestampField [%s]: only identifiers like [id] or [a.id] are allowed", task.BiDirTimestampField)
	}

	if strings.TrimSpace(task.DataSQL) == "" {
		return errorx.NewBiz("data sql is required")
	}

	// 先按方言切割器切分，保证仅一条语句（解析器只解析单条，多条语句的尾部可能被忽略）
	stmtCount := 0
	if err := dialect.GetSQLSplitter().SplitSQL(strings.NewReader(task.DataSQL), func(s string) error {
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

	stmt, err := dialect.GetSQLParser().Parse(task.DataSQL)
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
		if !whereReg.MatchString(task.DataSQL) && dataSyncTrailingClauseReg.MatchString(task.DataSQL) {
			return errorx.NewBizf("data sql with with-clause cannot contain group by/having/order by/limit clauses when it has no where: appended where would produce invalid sql")
		}
		return nil
	default:
		return errorx.NewBizf("data sql must be a single select query, got [%s]", stmt.StmtKind())
	}
}

// validateCursorSemantics 校验 CursorInclusivity 与同步模式的组合合法性（交付语义显式）：
//
// “不重发”与“不漏行”不能同时失败：Inclusive 靠目标幂等吃下发重复行，幂等前提不成立时不得启用：
//   - Append(1) 无 UPSERT，目标行会默默重复 → 拒绝 Inclusive；
//   - FullRefresh(3) 不拼水位条件（needsFullScan）→ Inclusive 无意义也拒绝，避免无感配置；
//   - Validation(6) 不写目标 → Inclusive 无意义也拒绝；
//   - Auto 不限制（resolveCursorInclusivity 会按模式默认）。
func validateCursorSemantics(task *entity.DataSyncTask) error {
	if task.GetCursorInclusivity() != entity.CursorInclusivityInclusive {
		return nil
	}
	switch task.SyncMode {
	case entity.DataSyncModeIncrementalAppend:
		return errorx.NewBiz("cursorInclusivity=Inclusive is not allowed for Append mode: rows would be duplicated on the non-idempotent target; use Merge (UPSERT) or keep Exclusive")
	case entity.DataSyncModeFullRefresh, entity.DataSyncModeValidation:
		return errorx.NewBiz("cursorInclusivity=Inclusive has no effect for FullRefresh/Validation (they don't apply the watermark predicate); use Auto or Exclusive")
	}
	return nil
}

// dataSyncTrailingClauseReg WITH语句尾部子句关键字探测（词边界匹配，无法内省时的保守兜底）
var dataSyncTrailingClauseReg = regexp.MustCompile(`(?i)\b(group\s+by|having|order\s+by|limit|offset)\b`)

// validateDataSyncAppendable 校验Run时尾部拼接不会产生非法SQL。
//
// 同步执行按 `dataSQL [where 1=1] [and upd > val] [order by upd asc]` 形态在语句尾部追加条件：
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

// dataSQLHasWhere 判断DataSQL是否已含WHERE条件（用于决定是否补"where 1=1"）。
//
// 优先用源方言解析器判定（SelectStmt.Where非空），替代旧(?i)where正则——
// 旧正则会把不含where条件但字段名/表名/字符串字面量含"where"子串的SQL误判为已有条件，
// 导致后续拼接"and updField > x"时产生非法SQL。
// WITH语句（解析结果无法内省where）与解析失败场景降级正则兜底
func dataSQLHasWhere(dialect dbi.Dialect, task *entity.DataSyncTask) bool {
	if dialect != nil {
		if stmt, err := dialect.GetSQLParser().Parse(task.DataSQL); err == nil {
			if sel, ok := stmt.(*sqlstmt.SelectStmt); ok {
				return sel.Where != nil
			}
			return whereReg.MatchString(task.DataSQL)
		}
	}
	return whereReg.MatchString(task.DataSQL)
}
