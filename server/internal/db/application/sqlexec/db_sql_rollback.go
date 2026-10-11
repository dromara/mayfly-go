package sqlexec

// 依据 SQL 执行记录采集到的旧值生成回滚 SQL。
//
// 三条边界决定这里只生成文本、不做执行：
//  1. 回滚是一次新的变更，必须走它自己的预检与审批；平台若开一条「回滚不过策略」的通道，
//     任何变更都能先斩后奏再拿回滚掩盖；
//  2. 旧值采集有行数上限（见 sqlExecAuditRows），截断时生成的语句只覆盖部分行，
//     必须把这件事写进返回里的 Note，不能让人以为「点了就能全改回去」；
//  3. 表结构在执行之后可能已经变了（列被删、主键被改）。此时按旧值硬拼会得到列值错位的语句，
//     宁可拒绝并说明原因，也不生成一条「看起来能跑」的错 SQL。
//
// 值字面量走各方言列类型的 Codec，而不是自己拼引号：旧值里的二进制列在 JSON 里是 hex 文本
// （经 dbi 的 ValuerBytes 归一），时间列是 'YYYY-MM-DD HH:MM:SS[.ffffff]' 文本（经 datetimeLayout
// 归一，非 RFC3339），整数列必须保住位数（超过 2^53 的 BIGINT 主键经 float64 会被改成邻近的数，
// 回滚 UPDATE 就落到错误的行或 0 行，而 0 行在界面上看起来像「执行成功」）

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/sqlutil"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
)

// sqlExecAuditRows 旧值采集的行数上限，采集侧（doUpdate/doDelete）与生成侧（截断说明）共用一个数：
// 两边各写一份常量，改了一边就会让「是否被截断」的判定静默失真
const sqlExecAuditRows = 200

// GenRollback 依据执行记录的旧值生成回滚 SQL。
//
// 只读：不改任何数据，也不执行生成的语句
func (d *DbSQLExecAppImpl) GenRollback(ctx context.Context, execRecordId uint64) (*dto.SqlRollback, error) {
	record, err := d.dbSQLExecRepo.GetById(execRecordId)
	if err != nil {
		return nil, err
	}
	if record == nil || record.Id == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackRecordNotFound, "id", execRecordId)
	}
	// 只为「确实执行成功过」的记录生成回滚：回滚是把旧值写回去的一次新变更，若这条记录对应的
	// 语句其实没生效（整批被回滚、或待执行/已取消的记录），照抄执行就会把陈旧值盖进现网数据，
	// 或把没删掉的行再插一遍。这类记录本身不该存在，但数据破坏只能靠这道闸拦在最后
	if record.Status != entity.DbSQLExecStatusSuccess {
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackNotExecuted, "id", execRecordId)
	}

	switch record.Type {
	case entity.DbSQLExecTypeUpdate, entity.DbSQLExecTypeDelete, entity.DbSQLExecTypeInsert:
	default:
		// 查询/DDL 谈不上回滚（DDL 的还原要反向 DDL，属结构对比的活，见后续计划）
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackUnsupportedType, "type", sqlExecTypeName(record.Type))
	}

	if strings.TrimSpace(record.Table) == "" {
		if record.Type == entity.DbSQLExecTypeInsert {
			// INSERT 表名为空说明当时语句未被解析，插入值快照也就无从采集
			return nil, errorx.NewBizI(ctx, imsg.ErrRollbackNoInsertSnapshot)
		}
		// 表名为空说明当时就没采到旧值（多表语句、无 WHERE、解析失败都会走到这里）
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackNoOldValue)
	}

	// 行数据源：UPDATE/DELETE 用执行前采集的旧值，INSERT 用解析出的插入值（存于 NewValue）
	rawRows := record.OldValue
	if record.Type == entity.DbSQLExecTypeInsert {
		rawRows = record.NewValue
	}
	rows, err := decodeOldValueRows(ctx, rawRows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		if record.Type == entity.DbSQLExecTypeInsert {
			return nil, errorx.NewBizI(ctx, imsg.ErrRollbackNoInsertSnapshot)
		}
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackNoOldValue)
	}

	conn, err := d.dbApp.GetDbConn(ctx, record.DbId, record.Db)
	if err != nil {
		return nil, err
	}

	table, err := loadRollbackColumns(ctx, conn, record.Table)
	if err != nil {
		return nil, err
	}

	result := &dto.SqlRollback{ExecId: record.Id, Type: int8(record.Type)}
	// 截断说明对三类行数据源都适用：采集上限之外的行无论哪类都回不来
	result.Note = rollbackTruncationNote(ctx, rows)

	switch record.Type {
	case entity.DbSQLExecTypeInsert:
		var conflicts []dto.SqlRollbackConflict
		result.Sqls, conflicts, err = genInsertRollback(ctx, conn, table, rows)
		if len(conflicts) > 0 {
			result.Conflicts = conflicts
			result.Note = mergeNote(result.Note, i18n.TC(ctx, imsg.RollbackBlockedNote, "count", len(conflicts)))
		}
	case entity.DbSQLExecTypeDelete:
		result.Sqls, err = rollbackInserts(ctx, conn, table, rows)
	default: // UPDATE
		outcome, uErr := genUpdateRollback(ctx, conn, table, record, rows)
		if uErr != nil {
			return nil, uErr
		}
		result.Verified = outcome.verified
		result.Conflicts = outcome.conflicts
		result.Note = mergeNote(result.Note, outcome.note)
		result.Sqls = outcome.sqls
	}
	if err != nil {
		return nil, err
	}
	result.Rows = len(result.Sqls)
	return result, nil
}

// mergeNote 追加一条说明；说明各自独立成句，用空格相接而不是覆盖——
// 截断、未校验、未解析等多条说明可能同时成立，覆盖会丢掉前一条
func mergeNote(base, add string) string {
	if add == "" {
		return base
	}
	if base == "" {
		return add
	}
	return base + " " + add
}

// updateRollbackOutcome UPDATE 回滚生成的完整结论。
//
// 收成方法返回值而非散装多值，是为了让 GenRollback 与集成测试走同一个入口、同一套判定：
// 「主键被改过必须拒绝」「无主键必须拒绝」这些守卫若只内联在 GenRollback 里，
// 集成测试直接调 rollbackUpdates 就会绕过它们——测试绿而生产路径的判定没人验
type updateRollbackOutcome struct {
	sqls      []string
	verified  bool
	conflicts []dto.SqlRollbackConflict
	note      string
}

// genUpdateRollback 生成一条 UPDATE 执行记录的回滚：先拦必须拒绝的情形（无主键、主键被改），
// 再做并发校验，最后按 CAS 生成语句。GenRollback 与集成测试共用此入口，判定只此一份
func genUpdateRollback(ctx context.Context, conn *dbi.DbConn, table *rollbackTable, record *entity.DbSQLExec, rows []map[string]any) (*updateRollbackOutcome, error) {
	if len(table.primaryKeys) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackNoPrimaryKey, "table", table.name)
	}

	outcome := &updateRollbackOutcome{}
	// 主键是否被改过只能看原语句的 SET 清单：旧值行里的键集合是「被改列+主键列」，
	// 主键同时出现在两边时从数据本身分不出来
	if setColumns, ok := updateSetColumns(conn, record.SQL); ok {
		for _, setColumn := range setColumns {
			if table.isPrimaryKey(setColumn) {
				return nil, errorx.NewBizI(ctx, imsg.ErrRollbackPkChanged, "column", setColumn)
			}
		}
	} else {
		// 解析不出原语句时如实告知「主键可能没被覆盖到」，而不是默默假定它没被改
		outcome.note = mergeNote(outcome.note, i18n.TC(ctx, imsg.RollbackStmtUnparsedNote))
	}

	verification, err := verifyUpdateRows(ctx, conn, table, rows, record.NewValue)
	if err != nil {
		return nil, err
	}
	outcome.verified = verification.verified
	if len(verification.conflicts) > 0 {
		outcome.conflicts = verification.conflicts
		outcome.note = mergeNote(outcome.note, i18n.TC(ctx, imsg.RollbackBlockedNote, "count", len(verification.conflicts)))
	}
	if !verification.verified {
		outcome.note = mergeNote(outcome.note, i18n.TC(ctx, imsg.RollbackUnverifiedNote))
	}
	outcome.sqls, err = rollbackUpdates(ctx, conn, table, verification.rows)
	if err != nil {
		return nil, err
	}
	return outcome, nil
}

// rollbackTruncationNote 旧值采集到过上限时给出如实说明：当时没记下的行谁也回不来。
//
// 采集侧（doDelete）是「到上限就停止扫描」而不是报错，所以旧值看着齐备、实际少行；
// 不说这一句，界面会把「生成了 N 条回滚 SQL」当成「全都能改回去」
func rollbackTruncationNote(ctx context.Context, rows []map[string]any) string {
	if len(rows) < sqlExecAuditRows {
		return ""
	}
	return i18n.TC(ctx, imsg.RollbackTruncatedNote, "cap", sqlExecAuditRows)
}

// rollbackInserts 为「被删除的行」生成 INSERT。
//
// 列顺序取表定义顺序而非行的 map 顺序：map 遍历无序，同一份旧值两次生成的 SQL 文本不同，
// 用户无法比对，也没法把这条语句沉淀成工单附件复核
func rollbackInserts(ctx context.Context, conn *dbi.DbConn, table *rollbackTable, rows []map[string]any) ([]string, error) {
	statements := make([]string, 0, len(rows))
	quote := conn.GetDialect().Quoter().QuoteIdent
	for _, row := range rows {
		columns, err := table.columnsOf(ctx, row)
		if err != nil {
			return nil, err
		}
		if len(columns) == 0 {
			continue
		}
		names := make([]string, 0, len(columns))
		values := make([]string, 0, len(columns))
		for _, column := range columns {
			names = append(names, quote(column.ColumnName))
			value, _ := LookupRowValue(row, column.ColumnName)
			values = append(values, rollbackLiteral(conn.Info.Type, column, value))
		}
		statements = append(statements, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(table.name), strings.Join(names, ", "), strings.Join(values, ", ")))
	}
	return statements, nil
}

// updateRollbackRow 一行 UPDATE 回滚的输入
type updateRollbackRow struct {
	// old 执行前采集的旧值：SET 的取值来源与主键定位来源
	old map[string]any
	// snapshot 执行后回查的行值：非 nil 表示该行做过并发校验、值进 CAS 条件；nil 表示未校验
	snapshot map[string]any
}

// updateVerification 并发校验结果
type updateVerification struct {
	// verified 是否真正做过比对（快照可用且回查成功）；false 表示降级生成，风险由生成侧的 Note 如实告知
	verified bool
	// rows 可生成回滚的行：校验通过的行带快照，取不到主键值无法配对的行原样透传（由生成侧报错拒绝）
	rows []updateRollbackRow
	// conflicts 校验不通过、不予生成的行
	conflicts []dto.SqlRollbackConflict
}

// rollbackUpdates 为「被改过的行」生成 UPDATE：SET 用非主键列的旧值，WHERE 用主键旧值。
//
// 校验通过的行把快照值追加进 WHERE 充当 CAS：这些列在「执行瞬间」的库中值只有快照说得准，
// 条件写回执行后值后，回滚落在与生成时相同的行状态上——若这期间有人又改了这些列，
// UPDATE 命中 0 行（宁可不还原，也不把并发修改悄悄盖掉）；
// 未校验的行（快照缺失的降级场景）只按主键定位，其风险由调用方在 Note 里如实说明
func rollbackUpdates(ctx context.Context, conn *dbi.DbConn, table *rollbackTable, rows []updateRollbackRow) ([]string, error) {
	if len(table.primaryKeys) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackNoPrimaryKey, "table", table.name)
	}

	statements := make([]string, 0, len(rows))
	quote := conn.GetDialect().Quoter().QuoteIdent
	for _, item := range rows {
		columns, err := table.columnsOf(ctx, item.old)
		if err != nil {
			return nil, err
		}

		sets := make([]string, 0, len(columns))
		for _, column := range columns {
			name := column.ColumnName
			if table.isPrimaryKey(name) {
				continue
			}
			value, _ := LookupRowValue(item.old, name)
			sets = append(sets, fmt.Sprintf("%s = %s", quote(name), rollbackLiteral(conn.Info.Type, column, value)))
		}
		if len(sets) == 0 {
			continue
		}

		conditions := make([]string, 0, len(table.primaryKeys)+len(columns))
		for _, pk := range table.primaryKeys {
			column, ok := table.column(pk)
			if !ok {
				return nil, errorx.NewBizI(ctx, imsg.ErrRollbackColumnGone, "column", pk, "table", table.name)
			}
			value, found := LookupRowValue(item.old, pk)
			if !found || value == nil {
				// 旧值里没有可用的主键就无法定位；宁可停下，也不生成一条会改到全表的 UPDATE
				return nil, errorx.NewBizI(ctx, imsg.ErrRollbackOldValueInvalid, "reason", fmt.Sprintf("the primary key column [%s] has no usable value in the recorded old values", pk))
			}
			conditions = append(conditions, rollbackEqCondition(quote(column.ColumnName), value, conn.Info.Type, column))
		}

		if item.snapshot != nil {
			// CAS 列取「快照里的非主键列」，与校验时的比较列同一集合：校验判过什么，条件就写什么
			for _, column := range table.order {
				if table.isPrimaryKey(column.ColumnName) {
					continue
				}
				value, found := LookupRowValue(item.snapshot, column.ColumnName)
				if !found {
					continue
				}
				conditions = append(conditions, rollbackEqCondition(quote(column.ColumnName), value, conn.Info.Type, column))
			}
		}
		statements = append(statements, fmt.Sprintf("UPDATE %s SET %s WHERE %s", quote(table.name), strings.Join(sets, ", "), strings.Join(conditions, " AND ")))
	}
	return statements, nil
}

// rollbackEqCondition 渲染等值条件：NULL 必须写 IS NULL——
// col = NULL 在 SQL 三元逻辑里恒为 UNKNOWN，条件永远不匹配，回滚会静默落 0 行
func rollbackEqCondition(quotedColumn string, value any, dbType dbi.DbType, column dbi.Column) string {
	if value == nil {
		return quotedColumn + " IS NULL"
	}
	return quotedColumn + " = " + rollbackLiteral(dbType, column, value)
}

// verifyUpdateRows 把旧值与执行后快照做并发比对：逐行按主键配对、一次回查拿回这些行的当前值，
// 与快照一致才放行生成（自那次变更之后没人动过这些列）。
//
// 三处降级/中断约定：
//  1. 快照缺失或解析失败 → 整体降级（verified=false）而不是拒绝生成：升级前的记录没有快照，
//     直接拒绝会让回滚功能对历史记录永远不可用；风险由生成侧的 Note 如实说明；
//  2. 回查失败 → 降级。不能把「查不到」当成「没改过」——那会把回查自身的故障伪装成
//     「全部校验通过」，是比不校验更危险的假安全感；
//  3. 行数据出现表已删掉的列 → 响亮报错，与生成侧同一口径
func verifyUpdateRows(ctx context.Context, conn *dbi.DbConn, table *rollbackTable, rows []map[string]any, snapshotRaw string) (*updateVerification, error) {
	degraded := func() *updateVerification {
		all := make([]updateRollbackRow, 0, len(rows))
		for _, row := range rows {
			all = append(all, updateRollbackRow{old: row})
		}
		return &updateVerification{rows: all}
	}

	snapshots, err := decodeOldValueRows(ctx, snapshotRaw)
	if err != nil {
		// 快照损坏不该阻断回滚：它与旧值同源同存，能坏说明记录本身异常，但旧值已解出可用
		logx.ErrorfContext(ctx, "rollback verify - failed to decode the post-execution snapshot: %s", err.Error())
		return degraded(), nil
	}
	if len(snapshots) == 0 {
		return degraded(), nil
	}

	// 快照按主键建索引；同一逻辑值在「存储解码」与「实时回查」两侧的文本形态不同
	// （json.Number 与 int64/string 等），统一经 canonicalValueText 归一后才能配对
	byKey := make(map[string]map[string]any, len(snapshots))
	for _, snapshot := range snapshots {
		key, _, ok := pkRowKey(table, snapshot)
		if !ok {
			continue
		}
		byKey[key] = snapshot
	}

	// 逐行校验旧值列完整性（列被删即停，与生成侧同一口径），并按能否取到主键值分成「可配对」与「配不上」
	type pendingRow struct {
		row map[string]any
		key string
	}
	paired := make([]pendingRow, 0, len(rows))
	unkeyed := make([]updateRollbackRow, 0)
	seen := make(map[string]bool)
	for _, row := range rows {
		if _, err := table.columnsOf(ctx, row); err != nil {
			return nil, err
		}
		for key := range row {
			seen[strings.ToLower(lastSegment(key))] = true
		}
		key, _, ok := pkRowKey(table, row)
		if !ok {
			unkeyed = append(unkeyed, updateRollbackRow{old: row})
			continue
		}
		paired = append(paired, pendingRow{row: row, key: key})
	}

	// 回查条件按旧主键值逐行 (pk=… AND …) 拼接，回查列取旧值里出现过的全部列（按表定义序）
	quote := conn.GetDialect().Quoter().QuoteIdent
	conditions := make([]string, 0, len(paired))
	for _, item := range paired {
		parts := make([]string, 0, len(table.primaryKeys))
		for _, pk := range table.primaryKeys {
			column, ok := table.column(pk)
			if !ok {
				return nil, errorx.NewBizI(ctx, imsg.ErrRollbackColumnGone, "column", pk, "table", table.name)
			}
			value, _ := LookupRowValue(item.row, pk) // pkRowKey 已保证存在且非 NULL
			parts = append(parts, rollbackEqCondition(quote(column.ColumnName), value, conn.Info.Type, column))
		}
		conditions = append(conditions, "("+strings.Join(parts, " AND ")+")")
	}
	selectColumns := make([]string, 0, len(table.order))
	for _, column := range table.order {
		if seen[strings.ToLower(column.ColumnName)] {
			selectColumns = append(selectColumns, quote(column.ColumnName))
		}
	}

	liveByKey := make(map[string]map[string]any, len(paired))
	if len(conditions) > 0 && len(selectColumns) > 0 {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(selectColumns, ", "), quote(table.name), strings.Join(conditions, " OR "))
		if _, err := conn.WalkQueryRows(ctx, query, func(live map[string]any, _ []*dbi.QueryColumn) error {
			key, _, ok := pkRowKey(table, live)
			if !ok {
				return nil
			}
			liveByKey[key] = live
			return nil
		}); err != nil {
			logx.ErrorfContext(ctx, "rollback verify - failed to re-read rows before generating: %s", err.Error())
			return degraded(), nil
		}
	}

	verification := &updateVerification{verified: true, rows: make([]updateRollbackRow, 0, len(rows))}
	for _, item := range paired {
		snapshot, hasSnapshot := byKey[item.key]
		live, hasLive := liveByKey[item.key]
		if !hasSnapshot || !hasLive {
			// 配对断了：行在快照里没有位置（当时回查就没看到它）或现在回查不到（已被删除）
			verification.conflicts = append(verification.conflicts, dto.SqlRollbackConflict{Key: rollbackKeyText(table, item.row), Reason: dto.RollbackConflictRowMissing})
			continue
		}
		if !updateRowUnchanged(table, snapshot, live) {
			verification.conflicts = append(verification.conflicts, dto.SqlRollbackConflict{Key: rollbackKeyText(table, item.row), Reason: dto.RollbackConflictConcurrentModified})
			continue
		}
		verification.rows = append(verification.rows, updateRollbackRow{old: item.row, snapshot: snapshot})
	}
	verification.rows = append(verification.rows, unkeyed...)
	return verification, nil
}

// updateRowUnchanged 快照与回查行是否一致：比较列取快照的非主键列（变更列），
// 两侧都归一到 JSON 文本再比较——同一逻辑值经同一套 dbi valuer 必得同一文本，
// 逐类型手写比较规则才是真正会把「等于」判错的地方
func updateRowUnchanged(table *rollbackTable, snapshot, live map[string]any) bool {
	columns := make([]string, 0, len(snapshot))
	for _, column := range table.order {
		if table.isPrimaryKey(column.ColumnName) {
			continue
		}
		if _, found := LookupRowValue(snapshot, column.ColumnName); found {
			columns = append(columns, column.ColumnName)
		}
	}
	return canonicalRowJSON(snapshot, columns) == canonicalRowJSON(live, columns)
}

// pkRowKey 取一行的主键配对键：各主键列的值文本以 \x00 连接（值本身含分隔符也不会串键）。
// 任一主键列缺失或为 NULL 都配不上（ok=false），badColumn 附带断点列
func pkRowKey(table *rollbackTable, row map[string]any) (key string, badColumn string, ok bool) {
	parts := make([]string, 0, len(table.primaryKeys))
	for _, pk := range table.primaryKeys {
		value, found := LookupRowValue(row, pk)
		if !found || value == nil {
			return "", pk, false
		}
		parts = append(parts, canonicalValueText(value))
	}
	return strings.Join(parts, "\x00"), "", true
}

// canonicalValueText 把值归一为配对文本：存储侧解码出 json.Number（保位数的数字），
// 回查侧来自驱动归一后的形态（int64/string/…），同值必须同文本才能配对
func canonicalValueText(value any) string {
	if number, ok := value.(json.Number); ok {
		return number.String()
	}
	return fmt.Sprintf("%v", value)
}

// rollbackKeyText 冲突行的可读标识：主键值逐列拼出（如 "id=2"），取不到主键则退化为 "pk=?"。
// 行标识本就在回滚 SQL 里出现，不构成额外暴露
func rollbackKeyText(table *rollbackTable, row map[string]any) string {
	parts := make([]string, 0, len(table.primaryKeys))
	for _, pk := range table.primaryKeys {
		value, found := LookupRowValue(row, pk)
		if !found {
			continue
		}
		if value == nil {
			parts = append(parts, pk+"=NULL")
			continue
		}
		parts = append(parts, pk+"="+canonicalValueText(value))
	}
	if len(parts) == 0 {
		return "pk=?"
	}
	return strings.Join(parts, ", ")
}

// genInsertRollback 为「被插入的行」生成 DELETE。
//
// INSERT 没有执行前的旧值可用——新行在执行前不存在，定位依据是语句解析出的插入值原文。
// 每行最多一条 DELETE，条件按值回放：行在插入后又被改过时条件匹配不上、语句命中 0 行，
// 宁可不删也不误删并发修改后的行
func genInsertRollback(ctx context.Context, conn *dbi.DbConn, table *rollbackTable, rows []map[string]any) ([]string, []dto.SqlRollbackConflict, error) {
	if len(table.primaryKeys) == 0 {
		return nil, nil, errorx.NewBizI(ctx, imsg.ErrRollbackNoPrimaryKey, "table", table.name)
	}

	quote := conn.GetDialect().Quoter().QuoteIdent
	statements := make([]string, 0, len(rows))
	conflicts := make([]dto.SqlRollbackConflict, 0)
	for i, row := range rows {
		statement, reason, err := buildInsertDeleteSQL(ctx, conn, table, row, quote)
		if err != nil {
			return nil, nil, err
		}
		if reason != "" {
			conflicts = append(conflicts, dto.SqlRollbackConflict{Key: fmt.Sprintf("row #%d", i+1), Reason: reason})
			continue
		}
		statements = append(statements, statement)
	}
	return statements, conflicts, nil
}

// buildInsertDeleteSQL 一行插入值快照 → 一条 DELETE；返回的 reason 非空表示该行定位失败。
//
// 快照单元格是插入语句里的字面量原文（字符串含引号）或 NULL，条件按原文回放：不解码再编码，
// \n、”、E”、q'[]' 等转义形态就不会被二次改写。主键列在插入列内且非 NULL 时条件已含主键，
// 直接按「主键 + 全部插入值」精确到行；主键缺位（自增主键未列入等）时先按插入值回查候选行——
// 回查到唯一一行才把它真正的主键补进条件，0 行/多行都停下：凭值删行在「本来就有同值行」
// 或「后来又插了同值行」时会删错对象
func buildInsertDeleteSQL(ctx context.Context, conn *dbi.DbConn, table *rollbackTable, row map[string]any, quote func(string) string) (string, string, error) {
	columns, err := table.columnsOf(ctx, row)
	if err != nil {
		return "", "", err
	}

	conditions := make([]string, 0, len(columns))
	for _, column := range columns {
		value, _ := LookupRowValue(row, column.ColumnName)
		if value == nil {
			conditions = append(conditions, quote(column.ColumnName)+" IS NULL")
			continue
		}
		text, ok := value.(string)
		if !ok {
			// 快照单元格只可能是字面量原文或 null；出现其他形态说明记录已损坏
			return "", "", errorx.NewBizI(ctx, imsg.ErrRollbackOldValueInvalid, "reason", fmt.Sprintf("unexpected value shape for column [%s] in the recorded insert values", column.ColumnName))
		}
		conditions = append(conditions, quote(column.ColumnName)+" = "+text)
	}
	if len(conditions) == 0 {
		return "", dto.RollbackConflictUnknownPosition, nil
	}

	// 主键齐备（在插入列内且非 NULL）：条件已含主键，直接生成
	pkReady := true
	for _, pk := range table.primaryKeys {
		value, found := LookupRowValue(row, pk)
		if !found || value == nil {
			pkReady = false
			break
		}
	}
	if pkReady {
		return fmt.Sprintf("DELETE FROM %s WHERE %s", quote(table.name), strings.Join(conditions, " AND ")), "", nil
	}

	// 主键缺位：按插入值回查候选行，命中第二行立即停（多行即无法唯一定位，不必读完）
	selectColumns := make([]string, 0, len(table.primaryKeys))
	for _, pk := range table.primaryKeys {
		column, ok := table.column(pk)
		if !ok {
			return "", "", errorx.NewBizI(ctx, imsg.ErrRollbackColumnGone, "column", pk, "table", table.name)
		}
		selectColumns = append(selectColumns, quote(column.ColumnName))
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(selectColumns, ", "), quote(table.name), strings.Join(conditions, " AND "))

	var (
		foundRow  map[string]any
		found     bool
		ambiguous bool
	)
	if _, err := conn.WalkQueryRows(ctx, query, func(live map[string]any, _ []*dbi.QueryColumn) error {
		if found {
			ambiguous = true
			return dbi.NewStopWalkQueryError("more than one row matches the recorded insert values")
		}
		foundRow = live
		found = true
		return nil
	}); err != nil {
		return "", "", err
	}
	if !found {
		return "", dto.RollbackConflictUnknownPosition, nil
	}
	if ambiguous {
		return "", dto.RollbackConflictAmbiguousMatch, nil
	}

	// 唯一候选：把回查到的真实主键值补进条件（与插入值条件一起构成 CAS，删除对象不因主键复用而漂移）
	pkConditions := make([]string, 0, len(table.primaryKeys))
	for _, pk := range table.primaryKeys {
		column, _ := table.column(pk) // 上面已校验存在
		value, ok := LookupRowValue(foundRow, pk)
		if !ok || value == nil {
			return "", dto.RollbackConflictUnknownPosition, nil
		}
		pkConditions = append(pkConditions, rollbackEqCondition(quote(column.ColumnName), value, conn.Info.Type, column))
	}
	return fmt.Sprintf("DELETE FROM %s WHERE %s", quote(table.name), strings.Join(append(pkConditions, conditions...), " AND ")), "", nil
}

// rollbackTable 生成回滚 SQL 所需的表侧事实：列信息（按表定义顺序）与主键列
type rollbackTable struct {
	name        string
	order       []dbi.Column
	byName      map[string]dbi.Column
	primaryKeys []string
}

// loadRollbackColumns 取表的列与主键。表读不到即拒绝生成：
// 拿不到列类型就写不出正确的字面量，凭旧值的 JSON 形态猜会把字符串塞进数值列
func loadRollbackColumns(ctx context.Context, conn *dbi.DbConn, tableName string) (*rollbackTable, error) {
	columns, err := conn.Metadata().GetColumns(tableName)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackNoTable, "table", tableName)
	}
	primaryKeys, err := conn.Metadata().GetPrimaryKeys(tableName)
	if err != nil {
		return nil, err
	}
	table := &rollbackTable{name: tableName, order: columns, byName: make(map[string]dbi.Column, len(columns)), primaryKeys: primaryKeys}
	for _, column := range columns {
		table.byName[strings.ToLower(column.ColumnName)] = column
	}
	return table, nil
}

// columnsOf 行里出现的列，按表定义顺序返回；旧值里有表已删掉的列即拒绝生成（列值会错位）。
//
// 行键先按「末段 + 小写」归一：驱动可能返回 `t.NAME` 这种带别名前缀或大小写不同的形态，
// 不归一就会把还存在的列错报成「已被删除」
func (t *rollbackTable) columnsOf(ctx context.Context, row map[string]any) ([]dbi.Column, error) {
	seen := make(map[string]bool, len(row))
	for key := range row {
		seen[strings.ToLower(lastSegment(key))] = true
	}
	columns := make([]dbi.Column, 0, len(row))
	for _, column := range t.order {
		if seen[strings.ToLower(column.ColumnName)] {
			columns = append(columns, column)
		}
	}
	// 排序后逐个校验：map 遍历无序，不排序会让同一份旧值在不同次报错给出不同列名
	for _, key := range slices.Sorted(maps.Keys(row)) {
		if _, ok := t.byName[strings.ToLower(lastSegment(key))]; !ok {
			return nil, errorx.NewBizI(ctx, imsg.ErrRollbackColumnGone, "column", displayRowKey(key), "table", t.name)
		}
	}
	return columns, nil
}

func (t *rollbackTable) column(name string) (dbi.Column, bool) {
	col, ok := t.byName[strings.ToLower(name)]
	return col, ok
}

func (t *rollbackTable) isPrimaryKey(name string) bool {
	for _, pk := range t.primaryKeys {
		if strings.EqualFold(pk, name) {
			return true
		}
	}
	return false
}

// updateSetColumns 从原 UPDATE 语句取被赋值的列；解析不出来返回 ok=false（不同方言的解析器
// 对同一条语句的把握不同，这里不猜，交由调用方如实标注）
func updateSetColumns(conn *dbi.DbConn, sql string) ([]string, bool) {
	stmt, err := conn.GetDialect().GetSQLParser().Parse(sql)
	if err != nil {
		return nil, false
	}
	updateStmt, ok := stmt.(*sqlstmt.UpdateStmt)
	if !ok {
		return nil, false
	}
	columns := make([]string, 0, len(updateStmt.Set))
	for _, assignment := range updateStmt.Set {
		// 归一为裸列名（去别名前缀）：isPrimaryKey 等消费方按裸名比对，SET x.id 改主键才不会被漏判
		if name := lastSegment(assignment.Column); name != "" {
			columns = append(columns, name)
		}
	}
	return columns, true
}

// decodeOldValueRows 解析执行记录里的旧值。
//
// 用 UseNumber 而不是默认的 float64：BIGINT 主键一旦经 float64 就会被改写，
// 生成的 WHERE 会落到别的行（或 0 行）
func decodeOldValueRows(ctx context.Context, raw string) ([]map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	rows := make([]map[string]any, 0, 8)
	if err := decoder.Decode(&rows); err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrRollbackOldValueInvalid, "reason", sqlutil.BriefReason(err))
	}
	return rows, nil
}

// LookupRowValue 按列名取旧值，容忍库侧返回的 `alias.col` 形态
func LookupRowValue(row map[string]any, column string) (any, bool) {
	want := strings.ToLower(column)
	for key, value := range row {
		if strings.ToLower(lastSegment(key)) == want {
			return value, true
		}
	}
	return nil, false
}

// lastSegment 去掉标识符里的限定前缀（schema./alias.）
func lastSegment(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// displayRowKey 报错里给用户的列名取末段，避免带着一串别名
func displayRowKey(key string) string {
	return lastSegment(key)
}

// rollbackLiteral 按目标库的列类型把旧值渲染成 SQL 字面量。
//
// 渲染策略只此一条：委托列类型的方言 Codec.SQLValue。旧值取自同一列的 dbi Valuer，Codec 是它的
// 对称逆操作——二进制列归一为 hex 文本、各方言 codec 渲染成 X'...' / 0x... / '\x...' 保真还原；
// 位列渲染成裸数字；布尔列渲染成各方言字面量。唯一的特殊处理是整数保 json.Number 原文（避免经
// float64 丢位：超过 2^53 的 BIGINT 主键一旦被改写，WHERE 就落到错误的行或 0 行）。
//
// 此处绝不能按类型类别另开渲染分支，也不能对 hex 文本做 base64 解码——两类「自作聪明」都曾造成
// 静默数据失真且语句仍能「执行成功」：base64 会把偶数字节的 hex 串（长度 4 的倍数）解成错误字节；
// 对 TCBool/TCBit 强走 SQLValueBool 会把 mysql BIT(8)=255 塌缩成 true(=1)。类别渲染是方言 Codec 的职责
func rollbackLiteral(dbType dbi.DbType, column dbi.Column, value any) string {
	if value == nil {
		return "NULL"
	}
	dataType := dbi.GetDbDataType(dbType, column.DataType)

	if number, ok := value.(json.Number); ok {
		// 保文本形态交给 codec，而不是先转成 float64：超过 2^53 的 BIGINT 主键一旦经 float64
		// 就会被改写成邻近的数，生成的 WHERE 落到错误的行或 0 行
		return dataType.Codec.SQLValue(number.String())
	}

	return dataType.Codec.SQLValue(value)
}

// sqlExecTypeName 执行类型的可读名称，用于「该类型不能回滚」的提示
func sqlExecTypeName(execType entity.DbSQLExecType) string {
	switch execType {
	case entity.DbSQLExecTypeInsert:
		return "INSERT"
	case entity.DbSQLExecTypeUpdate:
		return "UPDATE"
	case entity.DbSQLExecTypeDelete:
		return "DELETE"
	case entity.DbSQLExecTypeQuery:
		return "QUERY"
	case entity.DbSQLExecTypeDDL:
		return "DDL"
	case entity.DbSQLExecTypeOther:
		return "OTHER"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int8(execType))
	}
}
