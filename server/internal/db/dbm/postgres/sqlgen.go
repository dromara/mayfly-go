package postgres

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
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
		drops = append(drops, fmt.Sprintf("DROP INDEX IF EXISTS %s%s", currentSchema, index.IndexName))

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
			comments = append(comments, fmt.Sprintf("COMMENT ON INDEX %s%s IS '%s'", currentSchema, index.IndexName, comment))
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

func (psg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int) []string {
	insertSql := dbi.GenCommonInsert(psg.dialect, psg.dc.Info.Type, tableName, columns, values)

	// 根据冲突策略生成后缀
	suffix := ""
	if psg.dc.Info.Type == DbTypeGauss {
		// 高斯db使用ON DUPLICATE KEY UPDATE 语法参考 https://support.huaweicloud.com/distributed-devg-v3-gaussdb/gaussdb-12-0607.html#ZH-CN_TOPIC_0000001633948138
		suffix = psg.gaussOnDuplicateStrategySql(duplicateStrategy, tableName, columns)
	} else {
		// pgsql 默认使用 on conflict 语法参考 http://www.postgres.cn/docs/12/sql-insert.html
		// vastbase语法参考 https://docs.vastdata.com.cn/zh/docs/VastbaseE100Ver3.0.0/doc/SQL%E8%AF%AD%E6%B3%95/INSERT.html
		// kingbase语法参考 https://help.kingbase.com.cn/v8/development/sql-plsql/sql/SQL_Statements_9.html#insert
		suffix = psg.pgsqlOnDuplicateStrategySql(duplicateStrategy, tableName, columns)
	}

	return collx.AsArray[string](insertSql + suffix)
}

// pgsql默认唯一键冲突策略
func (psg *SQLGenerator) pgsqlOnDuplicateStrategySql(duplicateStrategy int, tableName string, columns []dbi.Column) string {
	suffix := ""
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		suffix = " \n on conflict do nothing"
	} else if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		// 生成 on conflict () do update set column1 = excluded.column1, column2 = excluded.column2, ...
		var updateColumns []string
		for _, col := range columns {
			updateColumns = append(updateColumns, fmt.Sprintf("%s = excluded.%s", col.ColumnName, col.ColumnName))
		}
		// 查询唯一键名,拼接冲突sql
		_, keyRes, _ := psg.dc.Query("SELECT constraint_name FROM information_schema.table_constraints WHERE constraint_schema = $1 AND table_name = $2 AND constraint_type in ('PRIMARY KEY', 'UNIQUE') ", psg.dc.Info.CurrentSchema(), tableName)
		if len(keyRes) > 0 {
			for _, re := range keyRes {
				key := anyx.ToString(re["constraint_name"])
				if key != "" {
					suffix += fmt.Sprintf(" \n on conflict on constraint %s do update set %s \n", key, strings.Join(updateColumns, ", "))
				}
			}
		}
	}
	return suffix
}

// 高斯db唯一键冲突策略,使用ON DUPLICATE KEY UPDATE 参考：https://support.huaweicloud.com/distributed-devg-v3-gaussdb/gaussdb-12-0607.html#ZH-CN_TOPIC_0000001633948138
func (psg *SQLGenerator) gaussOnDuplicateStrategySql(duplicateStrategy int, tableName string, columns []dbi.Column) string {
	suffix := ""
	metadata := psg.dc.GetMetadata()
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		suffix = " \n ON DUPLICATE KEY UPDATE NOTHING"
	} else if duplicateStrategy == dbi.DuplicateStrategyUpdate {

		// 查出表里的唯一键涉及的字段
		var uniqueColumns []string
		indexs, err := metadata.GetTableIndex(tableName)
		if err == nil {
			for _, index := range indexs {
				if index.IsUnique {
					cols := strings.Split(index.ColumnName, ",")
					for _, col := range cols {
						if !collx.ArrayContains(uniqueColumns, strings.ToLower(col)) {
							uniqueColumns = append(uniqueColumns, strings.ToLower(col))
						}
					}
				}
			}
		}

		suffix = " \n ON DUPLICATE KEY UPDATE "
		for i, col := range columns {
			// ON DUPLICATE KEY UPDATE语句不支持更新唯一键字段，所以得去掉
			if !collx.ArrayContains(uniqueColumns, psg.dialect.Quoter().Trim(strings.ToLower(col.ColumnName))) {
				suffix += fmt.Sprintf("%s = excluded.%s", col.ColumnName, col.ColumnName)
				if i < len(columns)-1 {
					suffix += ", "
				}
			}
		}
	}
	return suffix
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
	if column.IsIdentity {
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
			defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	columnSql := fmt.Sprintf(" %s %s%s%s", colName, column.GetColumnType(), nullAble, defVal)
	return columnSql
}
