package sqlite

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

var _ dbi.SQLGenerator = (*SQLGenerator)(nil)

type SQLGenerator struct {
	dbi.BaseSQLGenerator
	dialect dbi.Dialect
}

func (ssg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	quoter := ssg.dialect.Quoter()

	sqlArr := make([]string, 0)
	tbName := ssg.dialect.Quoter().QuoteIdent(table.TableName)
	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", tbName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", tbName)
	fields := make([]string, 0)

	// 把通用类型转换为达梦类型
	// 复合主键必须用表级 PRIMARY KEY(...) 声明：逐列内联 PRIMARY KEY 会生成非法DDL
	pkCount := 0
	for _, column := range columns {
		if column.IsPrimaryKey {
			pkCount++
		}
	}
	for _, column := range columns {
		fields = append(fields, ssg.genColumnBasicSql(quoter, column, pkCount == 1))
	}
	createSql += strings.Join(fields, ",\n")
	if pkCount > 1 {
		pkNames := make([]string, 0, pkCount)
		for _, column := range columns {
			if column.IsPrimaryKey {
				pkNames = append(pkNames, quoter.QuoteIdent(column.ColumnName))
			}
		}
		createSql += fmt.Sprintf(",\nPRIMARY KEY (%s)", strings.Join(pkNames, ","))
	}
	createSql += "\n)"

	sqlArr = append(sqlArr, createSql)

	return sqlArr
}

// sqliteImplicitIndexPrefix 约束（如UNIQUE）自动生成的隐式索引名前缀
const sqliteImplicitIndexPrefix = "sqlite_autoindex_"

// ddlIndexName 返回用于重建的索引名。
//
// sqlite_ 前缀为内核保留，以 sqlite_autoindex_* 原名执行CREATE INDEX会报
// “object name reserved for internal use”，且该索引不可被DROP，故迁移产物中以idx_别名重建
func ddlIndexName(indexName string) string {
	if rest := strings.TrimPrefix(indexName, sqliteImplicitIndexPrefix); rest != indexName && rest != "" {
		return "idx_" + rest
	}
	return indexName
}

func (ssg *SQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	quoter := ssg.dialect.Quoter()
	quote := quoter.QuoteIdent

	sqls := make([]string, 0)
	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}
		name := ddlIndexName(index.IndexName)
		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, col := range cols {
			colNames[i] = quote(col)
		}
		// 创建前尝试删除（索引名必须走quote，硬编码引号会让含双引号的名称提前闭合）
		sqls = append(sqls, fmt.Sprintf("DROP INDEX IF EXISTS %s", quote(name)))

		sqlTmp := "CREATE %s INDEX %s ON %s (%s) "
		sqls = append(sqls, fmt.Sprintf(sqlTmp, unique, quote(name), quote(table.TableName), strings.Join(colNames, ",")))
	}

	return sqls
}

func (ssg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {
	if duplicateStrategy == dbi.DuplicateStrategyNone {
		return collx.AsArray(dbi.GenCommonInsert(ssg.dialect, DbTypeSqlite, tableName, columns, values))
	}

	sqls := make([]string, 0)
	sqls = append(sqls, "PRAGMA foreign_keys = false")

	quote := ssg.dialect.Quoter().QuoteIdent

	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(ssg.dialect, DbTypeSqlite, columns, values)
		sqls = append(sqls, fmt.Sprintf("insert or ignore into %s %s VALUES \n%s", quote(tableName), columnStr, strings.Join(valuesStrs, ",\n")))
		sqls = append(sqls, "PRAGMA foreign_keys = true")
		return sqls
	}

	// DuplicateStrategyUpdate: 真正的 UPSERT（ON CONFLICT DO UPDATE SET），
	// 替代旧 INSERT OR REPLACE（本质 DELETE+INSERT，会破坏外键关系）
	// 需要 SQLite 3.35.0+ 支持 ON CONFLICT DO UPDATE 语法
	columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(ssg.dialect, DbTypeSqlite, columns, values)

	conflictClause := ssg.genOnConflictDoUpdate(columns, targetTableMeta)
	if conflictClause == "" {
		// 无法生成 UPDATE 子句，退化为 INSERT OR IGNORE
		sqls = append(sqls, fmt.Sprintf("insert or ignore into %s %s VALUES \n%s", quote(tableName), columnStr, strings.Join(valuesStrs, ",\n")))
	} else {
		sqls = append(sqls, fmt.Sprintf("insert into %s %s VALUES \n%s %s", quote(tableName), columnStr, strings.Join(valuesStrs, ",\n"), conflictClause))
	}

	sqls = append(sqls, "PRAGMA foreign_keys = true")
	return sqls
}

// genOnConflictDoUpdate 生成 SQLite ON CONFLICT(...) DO UPDATE SET 子句。
// 需要 TargetTableMeta 提供唯一列；无唯一列时返回空串
func (ssg *SQLGenerator) genOnConflictDoUpdate(columns []dbi.Column, targetTableMeta *dbi.TargetTableMeta) string {
	if targetTableMeta == nil || len(targetTableMeta.UniqueColumns) == 0 {
		return ""
	}
	trim := ssg.dialect.Quoter().Trim
	quote := ssg.dialect.Quoter().QuoteIdent

	uniqueSet := make(map[string]bool, len(targetTableMeta.UniqueColumns))
	for _, col := range targetTableMeta.UniqueColumns {
		uniqueSet[strings.ToLower(col)] = true
	}

	var updateColumns []string
	for _, col := range columns {
		if uniqueSet[strings.ToLower(trim(col.ColumnName))] {
			continue
		}
		if dbi.PreservableGeneratedColumn(col, DbTypeSqlite) || col.IsGenerated {
			continue
		}
		quotedName := quote(col.ColumnName)
		updateColumns = append(updateColumns, fmt.Sprintf("%s = excluded.%s", quotedName, quotedName))
	}
	if len(updateColumns) == 0 {
		return ""
	}

	quotedUniqueCols := make([]string, 0, len(targetTableMeta.UniqueColumns))
	for _, col := range targetTableMeta.UniqueColumns {
		quotedUniqueCols = append(quotedUniqueCols, quote(col))
	}
	return fmt.Sprintf("on conflict (%s) do update set %s",
		strings.Join(quotedUniqueCols, ", "), strings.Join(updateColumns, ", "))
}

// GenTruncate 生成清空表语句。SQLite 无 TRUNCATE，退化为 DELETE FROM
func (ssg *SQLGenerator) GenTruncate(tableName string) []string {
	quote := ssg.dialect.Quoter().QuoteIdent
	return []string{fmt.Sprintf("DELETE FROM %s", quote(tableName))}
}

func (ssg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column, inlinePk bool) string {
	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	quoteColumnName := quoter.QuoteIdent(column.ColumnName)

	// 如果是单列主键，则直接返回，不判断默认值（复合主键的列级不内联，由表级子句统一声明）
	if column.IsPrimaryKey && inlinePk {
		// AUTOINCREMENT 仅允许用于 INTEGER PRIMARY KEY；非自增主键保留原列类型（无条件强制 integer 会导致 text 主键等迁移 DDL 错误）
		if column.AutoIncrement {
			return fmt.Sprintf(" %s integer PRIMARY KEY AUTOINCREMENT%s", quoteColumnName, nullAble)
		}
		return fmt.Sprintf(" %s %s PRIMARY KEY%s", quoteColumnName, column.GetColumnType(), nullAble)
	}

	// 默认值统一由dbi按元数据形态判定：字面量重新转义引用、函数类跳过、可证裸值原样输出；
	// 旧实现“含左括号即视为函数而丢弃”会使 DEFAULT '(0)'、'unknown (pending)' 类默认值静默丢失；
	// 源库为MySQL 8.0时默认值不带引号，必须能按字面量重新引用，不能因形态陌生而丢弃
	defVal := dbi.GenColumnDefaultSqlOf(&column, column.DataType, dbi.QuoteEscape)

	return fmt.Sprintf(" %s %s%s%s", quoteColumnName, column.GetColumnType(), nullAble, defVal)
}
