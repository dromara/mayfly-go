package sqlite

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

var _ dbi.SQLGenerator = (*SQLGenerator)(nil)

type SQLGenerator struct {
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

func (ssg *SQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	quoter := ssg.dialect.Quoter()
	quote := quoter.QuoteIdent

	sqls := make([]string, 0)
	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}
		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, name := range cols {
			colNames[i] = quote(name)
		}
		// 创建前尝试删除（索引名必须走quote，硬编码引号会让含双引号的名称提前闭合）
		sqls = append(sqls, fmt.Sprintf("DROP INDEX IF EXISTS %s", quote(index.IndexName)))

		sqlTmp := "CREATE %s INDEX %s ON %s (%s) "
		sqls = append(sqls, fmt.Sprintf(sqlTmp, unique, quote(index.IndexName), quote(table.TableName), strings.Join(colNames, ",")))
	}

	return sqls
}

func (ssg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {
	if duplicateStrategy == dbi.DuplicateStrategyNone {
		return collx.AsArray(dbi.GenCommonInsert(ssg.dialect, DbTypeSqlite, tableName, columns, values))
	}

	sqls := make([]string, 0)
	sqls = append(sqls, "PRAGMA foreign_keys = false")

	prefix := "insert or ignore into"
	if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		prefix = "insert or replace into"
	}

	columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(ssg.dialect, DbTypeSqlite, columns, values)

	sqls = append(sqls, fmt.Sprintf("%s %s %s VALUES \n%s", prefix, ssg.dialect.Quoter().QuoteIdent(tableName), columnStr, strings.Join(valuesStrs, ",\n")))
	// 插入完成后再恢复外键约束（原实现将恢复语句置于 INSERT 之前，两条 PRAGMA 相邻执行互相抵消，INSERT 时外键约束仍处于开启状态）
	sqls = append(sqls, "PRAGMA foreign_keys = true")
	return sqls
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
