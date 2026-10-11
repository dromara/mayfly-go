package sqlexec

// 行级快照的采集与归一：UPDATE 的「执行后快照」与 INSERT 的「插入值快照」。
//
// UPDATE 的快照来自执行后按主键回查：SQL 文本里的 SET 推不出真实写入值（表达式、函数、
// 触发器、类型归一都会让「意图值」与「库中值」分叉），只有回查到的事实能作为并发校验基准
// 与 CAS 条件的值来源。
//
// INSERT 的快照来自语句解析出的 VALUES 字面量（字面量为空的形态由 sqlparser 整体放弃）：
// 新行没有可用的历史定位（自增主键执行前不存在），字面量是唯一线索。存储保留原文、
// 生成回滚条件时按原文回放，避免「解码再编码」对 \n、''、E'' 等转义形态的二次改写。

import (
	"context"
	"fmt"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/jsonx"
)

// insertSnapshotOf 把 INSERT 语句解析出的行字面量转成快照 JSON（列名 → 字面量原文，NULL 为 JSON null）。
//
// 超出 sqlExecAuditRows 只保留前若干行：截断事实由生成侧的说明如实告知（与旧值采集同一条上限）；
// 行列数不齐无法可靠配对，整体放弃而不是硬凑
func insertSnapshotOf(stmt *sqlstmt.InsertStmt) string {
	if len(stmt.Columns) == 0 || len(stmt.ValueRows) == 0 {
		return ""
	}
	rows := make([]map[string]any, 0, len(stmt.ValueRows))
	for _, valueRow := range stmt.ValueRows {
		if len(valueRow) != len(stmt.Columns) {
			return ""
		}
		if len(rows) >= sqlExecAuditRows {
			break
		}
		row := make(map[string]any, len(valueRow))
		for i, column := range stmt.Columns {
			if valueRow[i].Null {
				row[column] = nil
				continue
			}
			row[column] = valueRow[i].Text
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return ""
	}
	return jsonx.ToStr(rows)
}

// updateTouchesPrimaryKey SET 列是否触及任一主键列（大小写不敏感）
func updateTouchesPrimaryKey(updateColumns, primaryKeys []string) bool {
	for _, updateColumn := range updateColumns {
		for _, pk := range primaryKeys {
			if strings.EqualFold(updateColumn, pk) {
				return true
			}
		}
	}
	return false
}

// captureUpdatePostImage 在 UPDATE 执行成功后，按旧值里的主键逐行回查「变更列+主键」的当前值，
// 作为执行后快照（回滚生成的并发校验基准与 CAS 值来源）。返回空串表示未采到。
//
// 跳过的情形（无旧值、无主键、元数据/回查失败）一律不阻断执行：快照是回滚能力的增强
// 而不是执行的前置条件，为一次采样失败去拒绝用户的正常 UPDATE 是本末倒置；
// 生成侧对空快照的处理是降级生成 + 如实说明，链路依旧可用
func captureUpdatePostImage(ctx context.Context, p *sqlExecParam, oldRows []map[string]any, changeColumns, primaryKeys []string) string {
	if len(oldRows) == 0 || len(primaryKeys) == 0 {
		// 没有旧值就没有回查的主键集合；没有主键就无从定位执行后的行
		return ""
	}
	tableName := p.SQLExecRecord.Table
	columns, err := p.DbConn.Metadata().GetColumns(tableName)
	if err != nil {
		logx.ErrorfContext(ctx, "update post-image - failed to load columns of %s: %s", tableName, err.Error())
		return ""
	}
	byName := make(map[string]dbi.Column, len(columns))
	for _, column := range columns {
		byName[strings.ToLower(column.ColumnName)] = column
	}
	// 标识符一律按「目录真名」引用：列名可能是保留字（order/key/group）或大小写敏感（pg/oracle 的 UserId），
	// 裸拼会语法错或查不到；但直接引用语句里的书写形态同样错——折叠大小写的方言（pg）里书写形态与目录真名
	// 并不相等，会引用出一个不存在的列。两处都归一到 byName 这份目录索引（见 db_sql_ident.go）
	ident := newIdentQuoter(p.DbConn)
	columnCatalog, catalogTable := newColumnCatalog(columns)
	quote := ident.quote

	// 主键条件逐行渲染为 (pk1 = lit1 AND pk2 = lit2)：字面量走列类型的 Codec，
	// 大整数主键必须逐位保留（float64 会把 2^53 之外的数改成邻近值，回查到别的行或 0 行）
	conditions := make([]string, 0, len(oldRows))
	for _, row := range oldRows {
		parts := make([]string, 0, len(primaryKeys))
		for _, pk := range primaryKeys {
			column, ok := byName[strings.ToLower(pk)]
			if !ok {
				logx.ErrorfContext(ctx, "update post-image - primary key column %s not found in %s", pk, tableName)
				return ""
			}
			value, found := LookupRowValue(row, pk)
			if !found || value == nil {
				logx.ErrorfContext(ctx, "update post-image - primary key value of %s missing in captured old values", pk)
				return ""
			}
			parts = append(parts, fmt.Sprintf("%s = %s", quote(column.ColumnName), rollbackLiteral(p.DbConn.Info.Type, column, value)))
		}
		conditions = append(conditions, "("+strings.Join(parts, " AND ")+")")
	}

	// 回查列与原采集一致（变更列+主键）：快照与旧值同列集，生成侧才能逐列配对比较
	selectColumns := make([]string, 0, len(changeColumns)+len(primaryKeys))
	for _, name := range changeColumns {
		selectColumns = append(selectColumns, ident.catalogIdent(columnCatalog, name))
	}
	for _, name := range primaryKeys {
		// 主键名取自元数据，已是目录真名
		selectColumns = append(selectColumns, quote(name))
	}
	selectSQL := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(selectColumns, ","), ident.tableIdent(catalogTable, tableName), strings.Join(conditions, " OR "))

	snapshot := make([]map[string]any, 0, len(oldRows))
	if _, err := p.DbConn.WalkQueryRowsInTx(ctx, p.tx, selectSQL, func(row map[string]any, _ []*dbi.QueryColumn) error {
		snapshot = append(snapshot, row)
		if len(snapshot) >= sqlExecAuditRows {
			return dbi.NewStopWalkQueryError("reached the maximum number of post-image rows")
		}
		return nil
	}); err != nil {
		logx.ErrorfContext(ctx, "update post-image - failed to read rows after execution: %s", err.Error())
		return ""
	}
	return jsonx.ToStr(snapshot)
}

// canonicalRowJSON 把一行按给定列归一为稳定 JSON 文本：键序固定（json 对 map 键排序）、
// 大小写与别名前缀归一。
//
// 两侧取值都经过 dbi 同一套 Valuer（二进制=hex 文本、时间=字符串、超精度整数=字符串），
// 同一逻辑值在两次序列化下产出同一文本，因此比较可以退化为字符串相等，
// 不做逐类型比较——逐类型写比较规则才是真正会把「等于」判错的地方
func canonicalRowJSON(row map[string]any, columns []string) string {
	normalized := make(map[string]any, len(columns))
	for _, column := range columns {
		value, _ := LookupRowValue(row, column)
		normalized[strings.ToLower(column)] = value
	}
	return jsonx.ToStr(normalized)
}
