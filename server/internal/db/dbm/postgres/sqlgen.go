package postgres

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

type SQLGenerator struct {
	dialect dbi.Dialect

	dc *dbi.DbConn
}

func (msg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	quoter := msg.dialect.Quoter()
	quote := quoter.Quote
	quoteTableName := quote(table.TableName)

	sqlArr := make([]string, 0)
	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteTableName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", quoteTableName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)
	commentTmp := "COMMENT ON COLUMN %s.%s IS '%s'"

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, quote(column.ColumnName))
		}

		fields = append(fields, msg.genColumnBasicSql(quoter, column))

		// 防止注释内含有特殊字符串导致sql出错
		if column.ColumnComment != "" {
			comment := dbi.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf(commentTmp, quoteTableName, quote(column.ColumnName), comment))
		}
	}

	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	tableCommentSql := ""
	if table.TableComment != "" {
		commentTmp := "COMMENT ON TABLE %s IS '%s'"
		tableCommentSql = fmt.Sprintf(commentTmp, quoteTableName, dbi.QuoteEscape(table.TableComment))
	}

	// create
	sqlArr = append(sqlArr, createSql)

	// table comment
	if tableCommentSql != "" {
		sqlArr = append(sqlArr, tableCommentSql)
	}
	// column comment
	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}

	return sqlArr
}

func (msg *SQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	quoter := msg.dialect.Quoter()
	quote := quoter.Quote

	creates := make([]string, 0)
	drops := make([]string, 0)
	comments := make([]string, 0)
	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = " unique"
		}

		currentSchema := msg.dc.Info.CurrentSchema()
		// 带上后缀.  避免后续判断
		if currentSchema != "" {
			currentSchema = quote(currentSchema) + "."
		}

		// 如果索引名存在，先删除索引
		drops = append(drops, fmt.Sprintf("DROP INDEX IF EXISTS %s%s", currentSchema, quote(index.IndexName)))

		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, name := range cols {
			colNames[i] = quote(name)
		}
		// 创建索引
		creates = append(creates, fmt.Sprintf("CREATE%s INDEX %s ON %s%s(%s)", unique, quote(index.IndexName), currentSchema, quote(table.TableName), strings.Join(colNames, ",")))
		if index.IndexComment != "" {
			comment := dbi.QuoteEscape(index.IndexComment)
			comments = append(comments, fmt.Sprintf("COMMENT ON INDEX %s%s IS '%s'", currentSchema, quote(index.IndexName), comment))
		}
	}

	sqlArr := make([]string, 0)

	if len(drops) > 0 {
		sqlArr = append(sqlArr, drops...)
	}

	if len(creates) > 0 {
		sqlArr = append(sqlArr, creates...)
	}
	if len(comments) > 0 {
		sqlArr = append(sqlArr, comments...)
	}
	return sqlArr
}

func (psg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {
	insertSql := dbi.GenCommonInsert(psg.dialect, psg.dc.Info.Type, tableName, columns, values)

	// 根据冲突策略生成后缀
	suffix := ""
	if psg.dc.Info.Type == DbTypeGauss {
		// 高斯db使用ON DUPLICATE KEY UPDATE 语法参考 https://support.huaweicloud.com/distributed-devg-v3-gaussdb/gaussdb-12-0607.html#ZH-CN_TOPIC_0000001633948138
		suffix = psg.gaussOnDuplicateStrategySql(duplicateStrategy, targetTableMeta, columns)
	} else {
		// pgsql 默认使用 on conflict 语法参考 http://www.postgres.cn/docs/12/sql-insert.html
		// vastbase语法参考 https://docs.vastdata.com.cn/zh/docs/VastbaseE100Ver3.0.0/doc/SQL%E8%AF%AD%E6%B3%95/INSERT.html
		// kingbase语法参考 https://help.kingbase.com.cn/v8/development/sql-plsql/sql/SQL_Statements_9.html#insert
		suffix = psg.pgsqlOnDuplicateStrategySql(duplicateStrategy, targetTableMeta, columns)
	}

	return collx.AsArray[string](insertSql + suffix)
}

// pgsql默认唯一键冲突策略，生成过程中不查询数据库（唯一列由调用方预查传入）
func (psg *SQLGenerator) pgsqlOnDuplicateStrategySql(duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta, columns []dbi.Column) string {
	// on conflict do nothing 无需指定冲突列，可匹配任意唯一约束
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		return " \n on conflict do nothing"
	}
	if duplicateStrategy != dbi.DuplicateStrategyUpdate || targetTableMeta == nil || len(targetTableMeta.UniqueColumns) == 0 {
		return ""
	}

	// 生成 on conflict (uk_cols) do update set column1 = excluded.column1, ...
	// 注意：一条insert只能有一个on conflict子句，冲突列必须精确匹配某一个唯一约束的列集合，
	// 否则执行时报错，由调用方保证 UniqueColumns 为单一唯一约束（一般为表主键）的列集合
	uniqueSet := make(map[string]bool)
	for _, col := range targetTableMeta.UniqueColumns {
		uniqueSet[strings.ToLower(col)] = true
	}
	trim := psg.dialect.Quoter().Trim
	var updateColumns []string
	for _, col := range columns {
		// 不更新冲突键列自身，避免无意义更新
		if uniqueSet[strings.ToLower(trim(col.ColumnName))] {
			continue
		}
		updateColumns = append(updateColumns, fmt.Sprintf("%s = excluded.%s", col.ColumnName, col.ColumnName))
	}
	if len(updateColumns) == 0 {
		// 所有列均为冲突键列，无法生成update子句，退化为忽略冲突
		return " \n on conflict do nothing"
	}
	quote := psg.dialect.Quoter().Quote
	quotedUniqueCols := make([]string, 0, len(targetTableMeta.UniqueColumns))
	for _, col := range targetTableMeta.UniqueColumns {
		quotedUniqueCols = append(quotedUniqueCols, quote(col))
	}
	return fmt.Sprintf(" \n on conflict (%s) do update set %s \n", strings.Join(quotedUniqueCols, ", "), strings.Join(updateColumns, ", "))
}

// 高斯db唯一键冲突策略,使用ON DUPLICATE KEY UPDATE 参考：https://support.huaweicloud.com/distributed-devg-v3-gaussdb/gaussdb-12-0607.html#ZH-CN_TOPIC_0000001633948138
func (psg *SQLGenerator) gaussOnDuplicateStrategySql(duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta, columns []dbi.Column) string {
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		return " \n ON DUPLICATE KEY UPDATE NOTHING"
	}
	if duplicateStrategy != dbi.DuplicateStrategyUpdate || targetTableMeta == nil || len(targetTableMeta.UniqueColumns) == 0 {
		return ""
	}

	suffix := " \n ON DUPLICATE KEY UPDATE "
	trim := psg.dialect.Quoter().Trim
	var sets []string
	for _, col := range columns {
		// ON DUPLICATE KEY UPDATE语句不支持更新唯一键字段，所以得去掉
		if collx.ArrayContains(targetTableMeta.UniqueColumns, strings.ToLower(trim(col.ColumnName))) {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = excluded.%s", col.ColumnName, col.ColumnName))
	}
	if len(sets) == 0 {
		// 所有列均为唯一键列，无法生成update子句，退化为忽略冲突
		return ""
	}
	return suffix + strings.Join(sets, ", ")
}

func (pd *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	colName := quoter.Quote(column.ColumnName)
	dataType := string(column.DataType)

	// 如果数据类型是数字，则去掉长度
	if collx.ArrayAnyMatches([]string{"int"}, strings.ToLower(dataType)) {
		column.NumPrecision = 0
		column.CharMaxLength = 0
	}

	// 如果是自增类型，需要转换为serial
	if column.AutoIncrement {
		if dataType == "int4" {
			column.DataType = "serial"
		} else if dataType == "int2" {
			column.DataType = "smallserial"
		} else if dataType == "int8" {
			column.DataType = "bigserial"
		} else {
			column.DataType = "bigserial"
		}

		return fmt.Sprintf(" %s %s NOT NULL", colName, column.GetColumnType())
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		mark := false
		// 哪些字段类型默认值需要加引号
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, dataType) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) &&
				collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
		}
		// 如果数据类型是日期时间，则写死默认值函数
		if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) {
			column.ColumnDefault = "CURRENT_TIMESTAMP"
		}

		if mark {
			// 默认值可能含单引号（如 it's），需双写转义，避免 DDL 语法错误或注入
			defVal = fmt.Sprintf(" DEFAULT '%s'", dbi.QuoteEscape(column.ColumnDefault))
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	columnSql := fmt.Sprintf(" %s %s%s%s", colName, column.GetColumnType(), nullAble, defVal)
	return columnSql
}
