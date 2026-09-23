package sync

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

// upsertProbeBatchSize 单次目标冲突键探测的等值条件数上限（防止单条 OR 条件过长）
const upsertProbeBatchSize = 100

// writeSplit 一批目标写入的真实行数拆分：新增 / 更新 / 被唯一约束忽略而未写入。
type writeSplit struct {
	inserts int
	updates int
	ignored int
}

// coarseWriteSplit 无法探测冲突键存量时的兜底口径：整批按策略名义计数，不做无依据的拆分。
func coarseWriteSplit(strategy, rows int) writeSplit {
	if strategy == dbi.DuplicateStrategyUpdate {
		return writeSplit{updates: rows}
	}
	return writeSplit{inserts: rows}
}

// probeWriteSplit 还原一批写入的新增/更新/忽略行数。
//
// 必须在写入目标库之前调用：写入完成后所有冲突键都已存在，再也无法区分新增与更新。
// 各方言 upsert 语句返回的受影响行数语义互不一致（MySQL 命中更新计 2、PG/ORACLE MERGE 计 1），
// 也不能拿来换算，唯一跨方言可靠的依据是「写入前该冲突键是否已存在」。
//
// 无冲突键（GenInsert 退化为直接插入）或直接插入策略时不存在更新分支，整批即新增，不发探测查询。
//
// 计数是尽力而安的观测值：探测与写入之间他端并发提交同一冲突键，该行的新增/更新归属会偏一行；
// 因此不得拿它做数据一致性判据（那靠数据校验模式与删除对账）。
func probeWriteSplit(ctx context.Context, sec *syncExecContext, targetData []map[string]any) (writeSplit, error) {
	total := len(targetData)
	if total == 0 {
		return writeSplit{}, nil
	}
	strategy := cmp.Or(sec.task.DuplicateStrategy, dbi.DuplicateStrategyNone)
	// 未内省目标表元信息时无从判断冲突键，与无冲突键同义：退化为直接插入
	var uniqueCols []string
	if sec.targetTableMeta != nil {
		uniqueCols = sec.targetTableMeta.UniqueColumns
	}
	if strategy == dbi.DuplicateStrategyNone || len(uniqueCols) == 0 {
		return writeSplit{inserts: total}, nil
	}

	conn := sec.targetConn
	if !hasUpsertUpdateBranch(sec.targetColumns, uniqueCols) {
		// 目标列里没有一列既非冲突键又非生成列时，各方言都生成不出更新分支（退化为 do nothing / 直接插入），
		// 已存在的键不会被更新，故按忽略策略计数
		strategy = dbi.DuplicateStrategyIgnore
	}
	stat := collectKeyProbeClauses(targetData, uniqueCols, conn.GetDialect().Quoter().QuoteIdent, escapeFnForDialect(conn.Info.Type))
	existing, err := stat.countExistingKeys(ctx, conn, sec.task.TargetTableName)
	if err != nil {
		return writeSplit{}, err
	}
	if strategy == dbi.DuplicateStrategyIgnore {
		inserts, ignored := stat.splitIgnore(existing)
		return writeSplit{inserts: inserts, ignored: ignored}, nil
	}
	inserts, updates := stat.splitUpsert(existing)
	return writeSplit{inserts: inserts, updates: updates}, nil
}

// hasUpsertUpdateBranch 判断 upsert 语句能否形成更新分支：至少存在一个既非冲突键也非生成列的目标列。
// 与方言侧 SET 子句的列排除规则一致（见 mysql/postgres sqlgen 的冲突策略生成），无可选列时方言会把
// 更新策略退化为忽略插入，此时把已存在键的行计为更新便是假数。
func hasUpsertUpdateBranch(columns []dbi.Column, uniqueCols []string) bool {
	uniqueSet := make(map[string]struct{}, len(uniqueCols))
	for _, col := range uniqueCols {
		uniqueSet[strings.ToLower(col)] = struct{}{}
	}
	for _, column := range columns {
		if _, ok := uniqueSet[strings.ToLower(column.ColumnName)]; ok {
			continue
		}
		if column.IsGenerated {
			continue
		}
		return true
	}
	return false
}

// keyProbe 一批目标行去重后的冲突键探测集合。
type keyProbe struct {
	clauses []string // 去重后的冲突键等值条件，形如 ("id" = 1 AND "org" = 'a')
	keyed   int      // 具备完整非空冲突键的行数
	total   int      // 本批行数
}

// collectKeyProbeClauses 汇总一批目标行的冲突键。
// 冲突键列未映射或值为 NULL 的行不可能命中唯一约束（必然走插入分支），不计入 keyed。
func collectKeyProbeClauses(rows []map[string]any, uniqueCols []string, quote func(string) string, escapeFn func(any) string) keyProbe {
	stat := keyProbe{total: len(rows), clauses: make([]string, 0, len(rows))}
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		clause := uniqueKeyClause(row, uniqueCols, quote, escapeFn)
		if clause == "" {
			continue
		}
		stat.keyed++
		if _, ok := seen[clause]; ok {
			continue
		}
		seen[clause] = struct{}{}
		stat.clauses = append(stat.clauses, clause)
	}
	return stat
}

// uniqueKeyClause 构建单行在目标表的冲突键等值条件；任一冲突键缺失或为 NULL 时返回空串。
func uniqueKeyClause(row map[string]any, uniqueCols []string, quote func(string) string, escapeFn func(any) string) string {
	if len(uniqueCols) == 0 {
		return ""
	}
	conditions := make([]string, 0, len(uniqueCols))
	for _, col := range uniqueCols {
		val, ok := lookupRowValue(row, col)
		if !ok || val == nil {
			return ""
		}
		conditions = append(conditions, fmt.Sprintf("%s = %s", quote(col), formatSQLLiteral(val, escapeFn)))
	}
	return "(" + strings.Join(conditions, " AND ") + ")"
}

// countExistingKeys 统计探测条件中已在目标表存在的键数（分批探测，避免单条 OR 过长）。
// 目标唯一约束被破坏（同键多行）时每键仍按 1 计，否则更新行数会超过本批行数。
func (k keyProbe) countExistingKeys(ctx context.Context, conn *dbi.DbConn, tableName string) (int, error) {
	table := quoteTargetTableRef(conn, tableName)
	existing := 0
	for start := 0; start < len(k.clauses); start += upsertProbeBatchSize {
		end := min(start+upsertProbeBatchSize, len(k.clauses))
		probeSQL := fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s WHERE %s", table, strings.Join(k.clauses[start:end], " OR "))
		if _, err := conn.WalkQueryRows(ctx, probeSQL, func(row map[string]any, _ []*dbi.QueryColumn) error {
			existing += cast.ToInt(row["cnt"])
			return nil
		}); err != nil {
			return 0, fmt.Errorf("failed to probe the existing conflict keys of target table [%s]: %w", tableName, err)
		}
	}
	return min(existing, len(k.clauses)), nil
}

// splitUpsert upsert（更新策略）行的归属：已有冲突键的行为更新；同批内重复键的首次写入决定其归属，
// 后续重复写入必然命中该行，故同样计为更新。
func (k keyProbe) splitUpsert(existing int) (inserts, updates int) {
	updates = existing + (k.keyed - len(k.clauses))
	return k.total - updates, updates
}

// splitIgnore 忽略策略行的归属：冲突键已存在的行、以及同批内与之重复的行都被唯一约束丢弃，计为未写入。
func (k keyProbe) splitIgnore(existing int) (inserts, ignored int) {
	inserts = k.total - k.keyed + (len(k.clauses) - existing)
	return inserts, k.keyed - (len(k.clauses) - existing)
}
