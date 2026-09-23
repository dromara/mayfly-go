package sync

import (
	"fmt"
	"strings"
	"time"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

// 同步模块的批次/日志/命名辅助函数。

func normalizeBatchSize(size int) int {
	if size <= 0 {
		return defaultSyncBatchSize
	}
	return size
}

// bindSyncValues 将扫描中间值按目标列 codec 归一为驱动可绑定的 Go 值。
//
// 应用层不枚举 Category、也不判断方言：归一规则集中于 dbi.ValueCodec.BindValue 契约，
// 与文本路径的 SQLValue 同构（同一中间值经两条路径写入目标列结果一致）。
// 新方言引入新的不兼容中间值形态时，在自家 column.go 里 `.Copy().WithBindValue(...)` 覆写，
// 不回到本函数里堆叠方言/类型分支（开闭原则）。
func bindSyncValues(dbType dbi.DbType, columns []dbi.Column, rows [][]any) ([][]any, error) {
	bound := make([][]any, len(rows))
	for i, row := range rows {
		if len(row) != len(columns) {
			return nil, fmt.Errorf("sync row has %d values for %d columns", len(row), len(columns))
		}
		bound[i] = make([]any, len(columns))
		for j, column := range columns {
			codec := dbi.GetDbDataType(dbType, column.DataType).Codec
			v, err := codec.Bind(row[j])
			if err != nil {
				return nil, fmt.Errorf("column %q: %w", column.ColumnName, err)
			}
			bound[i][j] = v
		}
	}
	return bound, nil
}

// appendRunLog 向运行日志追加一行带时间戳的记录，并同步更新缓存以供前端实时查看。
// 格式：[HH:MM:SS.mmm] 消息内容\n
func (app *DataSyncAppImpl) appendRunLog(syncLog *entity.DataSyncLog, format string, args ...any) {
	if syncLog == nil {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] %s\n", ts, fmt.Sprintf(format, args...))
	syncLog.RunLog += line
	// 同步更新缓存，供 GetLogPageList 实时读取
	if syncLog.Id > 0 {
		app.setRunningSyncLog(syncLog.Id, syncLog)
	}
}

// getSyncModeName 返回同步模式的可读名称（仅用于 RunLog 日志输出，非前端展示）
func getSyncModeName(mode entity.DataSyncMode) string {
	switch mode {
	case entity.DataSyncModeFullRefresh:
		return "全量刷新"
	case entity.DataSyncModeIncrementalAppend:
		return "增量追加"
	case entity.DataSyncModeIncrementalMerge:
		return "增量合并"
	case entity.DataSyncModeIncrementalSoftDel:
		return "全量对账（源软删除）"
	case entity.DataSyncModeIncrementalHardDel:
		return "全量对账（源缺失删除）"
	case entity.DataSyncModeValidation:
		return "数据校验"
	default:
		return fmt.Sprintf("未知模式(%d)", mode)
	}
}

// getDuplicateStrategyName 返回冲突策略的可读名称（仅用于 RunLog 日志输出，非前端展示）
func getDuplicateStrategyName(strategy int) string {
	switch strategy {
	case dbi.DuplicateStrategyIgnore:
		return "忽略"
	case dbi.DuplicateStrategyUpdate:
		return "更新"
	default:
		return fmt.Sprintf("未知策略(%d)", strategy)
	}
}

// truncateString 截断字符串，超过长度时添加省略号。
// 按 rune 计数截断，避免切断 UTF-8 多字节字符产生乱码（日志中常含中文表名/值）。
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// quoteTargetTableRef 生成目标表在手写语句（冲突键探测/删除对账/数据校验）中的引用形式。
// mssql 表实际所属架构可能与连接默认架构不同，需带架构前缀（与该方言 GenInsert 的取名方式一致），
// 否则手写语句与生成语句指向的不是同一张表。
func quoteTargetTableRef(conn *dbi.DbConn, tableName string) string {
	quote := conn.GetDialect().Quoter().QuoteIdent
	table := quote(tableName)
	if conn.Info.Type == "mssql" {
		if schema := conn.Info.CurrentSchema(); schema != "" {
			return quote(schema) + "." + table
		}
	}
	return table
}

// BuildTargetTableMeta 构建目标表元信息，用于生成 upsert/merge 类插入语句
// （预先内省好并传入，保证后续 SQLGenerator 生成 SQL 的过程中不再查询数据库）。
// 冲突检测列优先取表主键；无主键且仅存在唯一一个唯一索引时取该索引列；否则 UniqueColumns 为空，
// GenInsert 退化为直接插入。
//
// 此「选取哪些列作为 upsert 冲突键」是数据同步的业务策略而非数据库内核能力，故置于应用层；
// dbi 仅保留承载结果的 TargetTableMeta 结构与 GenInsert(..., *TargetTableMeta) 契约。
func BuildTargetTableMeta(targetConn *dbi.DbConn, tableName string, columns []dbi.Column) *dbi.TargetTableMeta {
	meta := &dbi.TargetTableMeta{}
	var uniqueCols []string
	var identityCols []string
	for _, column := range columns {
		if column.IsPrimaryKey {
			uniqueCols = append(uniqueCols, column.ColumnName)
		}
		if column.AutoIncrement {
			identityCols = append(identityCols, column.ColumnName)
		}
	}
	if len(uniqueCols) == 0 {
		// 无主键时尝试取唯一索引，且仅当只存在一个唯一索引时才可作为冲突检测列（多个唯一索引无法确定冲突语义）
		indexes, err := targetConn.Metadata().GetTableIndex(tableName)
		if err == nil {
			var uniqueIndexes []dbi.Index
			for _, index := range indexes {
				if index.IsUnique {
					uniqueIndexes = append(uniqueIndexes, index)
				}
			}
			if len(uniqueIndexes) == 1 {
				for _, col := range strings.Split(uniqueIndexes[0].ColumnName, ",") {
					trimmed := strings.TrimSpace(col)
					if trimmed != "" {
						uniqueCols = append(uniqueCols, trimmed)
					}
				}
			}
		}
	}
	meta.UniqueColumns = uniqueCols
	meta.IdentityColumns = identityCols
	return meta
}
