package export

// Exporter 共享数据导出编排器（方言无关、格式无关）：
//
// 与 Consumer（格式层）构成「编排 + 插件」拆分：编排器负责游标遍历、分批、
// 列映射等共享流程，格式差异全部下沉到 Consumer 插件。
//
// 职责：
//   - DDL 导出（仅 SQL 格式）
//   - 数据导出（委托给 Consumer）
//   - 索引导出（仅 SQL 格式）
//   - 游标遍历 + 列映射 + 分批策略（行数+字节数双预算）
//
// 新增导出格式仅需实现 Consumer 接口并注册，本编排器无需任何修改（开闭原则）。
// 新增方言仅需实现 DumpTxnWrapper + SQLGenerator，导出功能自动可用。

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/writerx"
)

// Exporter 共享导出编排器。
type Exporter struct {
	dbConn   *dbi.DbConn
	consumer Consumer
	settings *Settings
	writer   io.Writer

	dumpDDL  bool
	dumpData bool

	tables      []string
	tableFilter map[string]string
	targetType  dbi.DbType

	// dropBeforeCreate 生成建表 DDL 前是否 DROP 同名表（默认 true，兼容既有行为）
	dropBeforeCreate bool
	// nameCase 目标对象名大小写转换策略（NameCaseNone/Upper/Lower），仅影响生成的 DDL/DML 目标标识符
	nameCase int

	log      func(string)
	progress func(string, dbi.DumpKind, int, bool)
}

// 目标对象名大小写转换策略（对应迁移任务 nameCase 配置）
const (
	// NameCaseNone 不转换（默认，0 亦按不转换处理）
	NameCaseNone = 1
	// NameCaseUpper 目标表名/列名转大写
	NameCaseUpper = 2
	// NameCaseLower 目标表名/列名转小写
	NameCaseLower = 3
)

// NewExporter 创建共享导出编排器。
func NewExporter(dbConn *dbi.DbConn, consumer Consumer, settings *Settings, writer io.Writer) *Exporter {
	if settings == nil {
		settings = DefaultSettings(consumer.Format())
	}
	return &Exporter{
		dbConn:           dbConn,
		consumer:         consumer,
		settings:         settings,
		writer:           writer,
		dropBeforeCreate: true, // 默认建表前 DROP，保持与既有硬编码行为一致
		nameCase:         NameCaseNone,
		log:              defaultLog,
		progress:         defaultProgress,
	}
}

// WithTables 设置要导出的表列表
func (e *Exporter) WithTables(tables []string) *Exporter {
	e.tables = tables
	return e
}

// WithTableFilter 设置表级数据过滤条件（表名→where条件）
func (e *Exporter) WithTableFilter(filter map[string]string) *Exporter {
	e.tableFilter = filter
	return e
}

// WithDumpDDL 设置是否导出 DDL
func (e *Exporter) WithDumpDDL(v bool) *Exporter {
	e.dumpDDL = v
	return e
}

// WithDumpData 设置是否导出数据
func (e *Exporter) WithDumpData(v bool) *Exporter {
	e.dumpData = v
	return e
}

// WithTargetType 设置目标数据库类型（跨方言导出时使用）
func (e *Exporter) WithTargetType(t dbi.DbType) *Exporter {
	e.targetType = t
	return e
}

// WithSkipDropTable 设置建表前是否**不**删除同名表（skip=true 时不生成 DROP，仅 DumpDDL 生效）。
// 默认 false（生成 DROP），保持与既有行为一致。
func (e *Exporter) WithSkipDropTable(skip bool) *Exporter {
	e.dropBeforeCreate = !skip
	return e
}

// WithNameCase 设置目标对象名大小写转换策略（NameCaseNone/Upper/Lower）。
// 仅影响生成的 DDL/DML 中的目标表名/列名，不影响源库查询与列映射。
func (e *Exporter) WithNameCase(nameCase int) *Exporter {
	e.nameCase = nameCase
	return e
}

// convIdent 按 nameCase 策略转换单个标识符（表名/列名）。
func (e *Exporter) convIdent(name string) string {
	return ConvertName(name, e.nameCase)
}

// ConvertName 是迁移输出与迁移校验共用的目标名称转换规则。
func ConvertName(name string, nameCase int) string {
	switch nameCase {
	case NameCaseUpper:
		return strings.ToUpper(name)
	case NameCaseLower:
		return strings.ToLower(name)
	default:
		return name
	}
}

// convTable 返回表名按 nameCase 转换后的 Table 副本（不修改入参，避免污染共享/缓存的元数据）。
func (e *Exporter) convTable(t dbi.Table) dbi.Table {
	if e.nameCase == NameCaseNone || e.nameCase == 0 {
		return t
	}
	cp := t
	cp.TableName = e.convIdent(t.TableName)
	return cp
}

// convColumns 返回列名/所属表名按 nameCase 转换后的列副本切片（深拷贝，保持顺序不变）。
// 行值按列顺序定位，故仅改列名不影响数据装配。
func (e *Exporter) convColumns(cols []dbi.Column) []dbi.Column {
	if e.nameCase == NameCaseNone || e.nameCase == 0 {
		return cols
	}
	cp := make([]dbi.Column, len(cols))
	for i, c := range cols {
		c.ColumnName = e.convIdent(c.ColumnName)
		c.TableName = e.convIdent(c.TableName)
		cp[i] = c
	}
	return cp
}

// convIndexes 返回索引列名按 nameCase 转换后的索引副本切片（深拷贝），使 CREATE INDEX 引用转换后的列。
func (e *Exporter) convIndexes(indexes []dbi.Index) []dbi.Index {
	if e.nameCase == NameCaseNone || e.nameCase == 0 {
		return indexes
	}
	cp := make([]dbi.Index, len(indexes))
	for i, idx := range indexes {
		idx.ColumnName = e.convIdent(idx.ColumnName)
		cp[i] = idx
	}
	return cp
}

// ValidateNameConversion 在输出任何 DDL 前检查名称碰撞及尚不支持的表达式改名。
// 生成列表达式属于源方言，不能通过整段大小写转换改写其中的字符串常量或函数。
func ValidateNameConversion(tables []string, columns []dbi.Column, nameCase int, targetType dbi.DbType) error {
	if nameCase == 0 || nameCase == NameCaseNone {
		return nil
	}
	seenTables := make(map[string]string)
	for _, table := range tables {
		name := ConvertName(table, nameCase)
		if previous, ok := seenTables[name]; ok && previous != table {
			return fmt.Errorf("name conversion maps tables %q and %q to %q", previous, table, name)
		}
		seenTables[name] = table
	}
	seenColumns := make(map[string]map[string]string)
	for _, column := range columns {
		if dbi.PreservableGeneratedColumn(column, targetType) {
			return fmt.Errorf("name conversion is not supported for generated column %q of table %q; use original names", column.ColumnName, column.TableName)
		}
		if seenColumns[column.TableName] == nil {
			seenColumns[column.TableName] = make(map[string]string)
		}
		seen := seenColumns[column.TableName]
		name := ConvertName(column.ColumnName, nameCase)
		if previous, ok := seen[name]; ok && previous != column.ColumnName {
			return fmt.Errorf("name conversion maps columns %q and %q of table %q to %q", previous, column.ColumnName, column.TableName, name)
		}
		seen[name] = column.ColumnName
	}
	return nil
}

// WithLog 设置日志回调
func (e *Exporter) WithLog(fn func(string)) *Exporter {
	if fn != nil {
		e.log = fn
	}
	return e
}

// WithProgress 设置进度回调
func (e *Exporter) WithProgress(fn func(string, dbi.DumpKind, int, bool)) *Exporter {
	if fn != nil {
		e.progress = fn
	}
	return e
}

// Export 执行完整导出。
func (e *Exporter) Export(ctx context.Context) error {
	writer := writerx.NewStringWriter(e.writer)
	dbName := e.dbConn.Info.Database

	// 写入导出头部注释（仅 SQL 格式）
	if e.consumer.SupportsScript() {
		header := strings.Join([]string{
			"\n-- ----------------------------",
			"\n-- Dump Platform: mayfly-go",
			fmt.Sprintf("\n-- Dump Time: %s ", time.Now().Format("2006-01-02 15:04:05")),
			fmt.Sprintf("\n-- Dump DB: %s ", dbi.SanitizeCommentText(dbName)),
			fmt.Sprintf("\n-- DB Dialect: %s ", cmp.Or(e.targetType, dbi.DbType(e.dbConn.Info.Type))),
			"\n-- ----------------------------\n\n",
		}, "")
		if err := writeString(writer, header); err != nil {
			return err
		}
	}

	// 获取目标方言（跨方言导出时使用目标方言生成 SQL）
	targetDialect := e.dbConn.GetDialect()
	if e.targetType != "" && e.dbConn.Info.Type != e.targetType {
		targetDialect = dbi.GetDialect(e.targetType)
	}

	srcMeta := e.dbConn.Metadata()
	srcDialect := e.dbConn.GetDialect()

	// 获取表列表
	tables := e.tables
	if len(tables) == 0 {
		e.log("gets the table information that can be export...")
		ti, err := srcMeta.GetTables()
		if err != nil {
			return err
		}
		tables = make([]string, len(ti))
		for i, t := range ti {
			tables[i] = t.TableName
		}
		e.log(fmt.Sprintf("Get %d tables", len(tables)))
	}
	if len(tables) == 0 {
		e.log("no table to export. end export")
		return errorx.NewBiz("there is no table to export")
	}
	// 注入表总数供格式自适应产物结构（如 JSON 单表数组/多表对象），必须在任何 Begin 之前
	e.settings.TableCount = len(tables)

	// 查询列信息
	e.log("querying column information...")
	columns, err := srcMeta.GetColumns(tables...)
	if err != nil {
		return err
	}

	if err := ValidateNameConversion(tables, columns, e.nameCase, cmp.Or(e.targetType, e.dbConn.Info.Type)); err != nil {
		return err
	}

	// 按表分组列信息，并转换为目标方言列类型
	columnMap := make(map[string][]dbi.Column)
	for _, column := range columns {
		if err := dbi.ConvToTargetDbColumn(e.dbConn.Info.Type, cmp.Or(e.targetType, dbi.DbType(e.dbConn.Info.Type)), targetDialect, &column); err != nil {
			return err
		}
		columnMap[column.TableName] = append(columnMap[column.TableName], column)
	}

	sort.Strings(tables)
	targetDumpTxnWrapper := targetDialect.GetDumpTxnWrapper()
	targetSQLGen := targetDialect.GetSQLGenerator()
	srcDialectQuote := srcDialect.Quoter().QuoteIdent

	// 逐表导出
	for _, tableName := range tables {
		e.log(fmt.Sprintf("get table [%s] information...", tableName))

		tbs, err := srcMeta.GetTables(tableName)
		if err != nil {
			return err
		}
		if len(tbs) <= 0 {
			return errorx.NewBizf("Failed to get table information: %s", tableName)
		}

		tableInfo := tbs[0]
		cols := columnMap[tableName]
		if len(cols) == 0 {
			return errorx.NewBizf("failed to get the columns information of the table [%s]", tableName)
		}

		// DDL 导出（仅 SQL 格式）
		if e.dumpDDL && e.consumer.SupportsScript() {
			if err := e.exportDDL(writer, tableName, tableInfo, cols, targetSQLGen); err != nil {
				return err
			}
		}

		// 数据导出（所有格式）
		if e.dumpData {
			if err := e.exportData(ctx, writer, tableName, cols, srcDialectQuote, targetDumpTxnWrapper, targetSQLGen); err != nil {
				return err
			}
		}

		// 索引导出（仅 SQL 格式，且仅在 DumpDDL 模式下）
		if e.dumpDDL && e.consumer.SupportsScript() {
			if err := e.exportIndex(writer, tableName, tableInfo, targetSQLGen); err != nil {
				return err
			}
		}
	}

	// 全局收尾：多表产物外层结构闭合（如 JSON 对象右括号），无此钩子则多表 JSON 为非法产物
	if err := e.consumer.Finish(writer, e.settings); err != nil {
		return err
	}

	return nil
}

// exportDDL 导出表结构 DDL
func (e *Exporter) exportDDL(writer *writerx.StringWriter, tableName string, tableInfo dbi.Table, cols []dbi.Column, sqlGen dbi.SQLGenerator) error {
	e.log(fmt.Sprintf("generate table [%s] DDL...", tableName))
	// DDL 使用目标侧标识符（按 nameCase 转换）；DROP 与否由 dropBeforeCreate 决定
	targetTable := e.convTable(tableInfo)
	targetCols := e.convColumns(cols)
	header := fmt.Sprintf("\n-- ----------------------------\n-- Table structure: %s \n-- ----------------------------\n", dbi.SanitizeCommentText(targetTable.TableName))
	if err := writeString(writer, header); err != nil {
		return err
	}
	ddlArr := sqlGen.GenTableDDL(targetTable, targetCols, e.dropBeforeCreate)
	for _, ddl := range ddlArr {
		if _, err := writer.WriteString(ddl + ";\n"); err != nil {
			return err
		}
	}
	e.progress(tableName, dbi.DumpKindDDL, len(ddlArr), true)
	return nil
}

// exportData 导出数据（共享编排逻辑：游标遍历 + 列映射 + 分批策略 + 消费者委托）
func (e *Exporter) exportData(ctx context.Context, writer *writerx.StringWriter, tableName string, cols []dbi.Column,
	srcDialectQuote func(string) string, helper dbi.DumpTxnWrapper, sqlGen dbi.SQLGenerator) error {

	e.log(fmt.Sprintf("generate table [%s] DML...", tableName))

	// 目标侧标识符（按 nameCase 转换）用于生成 INSERT 与方言前后置钩子；
	// 源库查询与列映射仍用原始标识符（源列名不受目标大小写策略影响）
	targetTableName := e.convIdent(tableName)
	targetCols := e.convColumns(cols)

	// 数据段头部注释（仅 SQL 格式）
	if e.consumer.SupportsScript() {
		header := fmt.Sprintf("\n-- ----------------------------\n-- Data: %s \n-- ----------------------------\n", dbi.SanitizeCommentText(targetTableName))
		if err := writeString(writer, header); err != nil {
			return err
		}
	}

	// 方言事务/前置钩子（仅 SQL 格式）
	if e.consumer.SupportsScript() {
		if err := helper.BeforeInsert(writer, targetTableName); err != nil {
			return err
		}
	}

	// 消费者开始（写格式头部，如 CSV 表头、JSON 数组起始符）
	if err := e.consumer.Begin(writer, targetTableName, targetCols, e.settings); err != nil {
		return err
	}

	// 分批缓冲区
	dataCount := 0
	rows := make([][]any, 0)
	pendingBytes := 0

	// flush 闭包：将缓冲区数据委托给消费者输出一批
	flushRows := func() error {
		if len(rows) == 0 {
			return nil
		}
		if err := e.consumer.ConsumeBatch(writer, targetTableName, targetCols, rows, helper, sqlGen, e.settings); err != nil {
			return err
		}
		rows = make([][]any, 0)
		pendingBytes = 0
		return nil
	}

	// 构建查询 SQL（支持表级过滤）
	srcQuoteTableName := srcDialectQuote(tableName)
	walkSQL := fmt.Sprintf("SELECT * FROM %s", srcQuoteTableName)
	if where := e.tableFilter[tableName]; where != "" {
		walkSQL += " WHERE " + where
	}

	// 列映射：首行时解析元数据列与查询结果列的对应关系
	var colKeys []string
	_, err := e.dbConn.WalkQueryRows(ctx, walkSQL, func(row map[string]any, queryCols []*dbi.QueryColumn) error {
		if colKeys == nil {
			resolvedKeys, err := resolveDumpColumnKeys(cols, queryCols)
			if err != nil {
				return err
			}
			colKeys = resolvedKeys
		}
		rowValues := make([]any, len(cols))
		rowBytes := 0
		for i, key := range colKeys {
			rowValues[i] = row[key]
			switch v := rowValues[i].(type) {
			case string:
				rowBytes += len(v)
			case []byte:
				rowBytes += len(v)
			default:
				rowBytes += 16
			}
		}
		rows = append(rows, rowValues)
		dataCount++
		pendingBytes += rowBytes

		// 行数与字节数双预算：达到任一预算即 flush
		if dataCount%DefaultBatchRows != 0 && pendingBytes < DefaultBatchBytes {
			return nil
		}
		if err := flushRows(); err != nil {
			return err
		}
		e.progress(tableName, dbi.DumpKindInsert, dataCount, false)
		return nil
	})
	if err != nil {
		return err
	}

	// flush 剩余数据
	if err := flushRows(); err != nil {
		return err
	}

	// 消费者结束（写格式尾部，如 JSON 数组闭合）
	if err := e.consumer.End(writer, targetTableName, e.settings); err != nil {
		return err
	}

	// 方言事务/后置钩子（仅 SQL 格式）
	if e.consumer.SupportsScript() {
		if err := helper.AfterInsert(writer, targetTableName, targetCols); err != nil {
			return err
		}
	}

	e.progress(tableName, dbi.DumpKindInsert, dataCount, true)
	return nil
}

// exportIndex 导出索引 DDL
func (e *Exporter) exportIndex(writer *writerx.StringWriter, tableName string, tableInfo dbi.Table, sqlGen dbi.SQLGenerator) error {
	srcMeta := e.dbConn.Metadata()
	indexes, err := srcMeta.GetTableIndex(tableName)
	if err != nil {
		return err
	}
	if len(indexes) == 0 {
		return nil
	}

	e.log(fmt.Sprintf("get table [%s] index information...", tableName))
	// 头部注释与 DDL/Data 段保持一致，使用目标侧（nameCase 转换后）表名
	header := fmt.Sprintf("\n-- ----------------------------\n-- Table Index: %s \n-- ----------------------------\n", dbi.SanitizeCommentText(e.convIdent(tableName)))
	if err := writeString(writer, header); err != nil {
		return err
	}

	e.log(fmt.Sprintf("generate table [%s] index...", tableName))
	// 索引 DDL 使用目标侧标识符（表名 + 索引列名按 nameCase 转换），与建表 DDL 保持一致
	sqlArr := sqlGen.GenIndexDDL(e.convTable(tableInfo), e.convIndexes(indexes))
	for _, sqlStr := range sqlArr {
		if _, err := writer.WriteString(sqlStr + ";\n"); err != nil {
			return err
		}
	}
	e.progress(tableName, dbi.DumpKindDDL, len(sqlArr), true)
	return nil
}

// writeString 写入文本，写入失败必须显式失败（避免静默截断）
func writeString(w *writerx.StringWriter, text string) error {
	_, err := w.WriteString(text)
	return err
}

// resolveDumpColumnKeys 解析元数据列与查询结果列的对应关系，返回每个元数据列在row map中的key。
//
// dump按列顺序组装INSERT的VALUES，若仅以元数据列名直接索引row，当后端返回的列标签与
// 元数据中的列名存在大小写/命名差异（如部分后端将列名转大写）时，索引不命中会静默得到nil，
// 导出为NULL——属不可逆的数据丢失，故此处先精确匹配、再大小写不敏感匹配，仍不命中则报错终止
func resolveDumpColumnKeys(columns []dbi.Column, queryCols []*dbi.QueryColumn) ([]string, error) {
	exact := make(map[string]string, len(queryCols))
	folded := make(map[string]string, len(queryCols))
	for _, qc := range queryCols {
		exact[qc.Name] = qc.Key
		folded[strings.ToLower(qc.Name)] = qc.Key
	}

	keys := make([]string, len(columns))
	for i, column := range columns {
		key, ok := exact[column.ColumnName]
		if !ok {
			key, ok = folded[strings.ToLower(column.ColumnName)]
		}
		if !ok {
			return nil, errorx.NewBizf("column [%s] of table [%s] not found in the query result columns, refuse to dump with unknown values",
				column.ColumnName, dbi.SanitizeCommentText(column.TableName))
		}
		keys[i] = key
	}
	return keys, nil
}

// defaultLog 默认日志回调：静默丢弃（编排器构造时的兜底，调用方可 WithLog 覆写）
func defaultLog(msg string) {
}

// defaultProgress 默认进度回调：静默丢弃
func defaultProgress(currentTable string, stmtType dbi.DumpKind, stmtCount int, currentStmtTypeEnd bool) {
}
