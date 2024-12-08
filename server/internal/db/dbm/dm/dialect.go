package dm

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
	"time"

	_ "gitee.com/chunanyong/dm"
)

type DMDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (dd *DMDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
	// 执行批量insert sql
	// insert into "table_name" ("column1", "column2", ...) values (value1, value2, ...)

	// 无需处理重复数据，直接执行批量insert
	if duplicateStrategy == dbi.DuplicateStrategyNone || duplicateStrategy == 0 {
		return dd.batchInsertSimple(tx, tableName, columns, values)
	} else { // 执行MERGE INTO语句
		return dd.batchInsertMerge(tx, tableName, columns, values)
	}
}

func (dd *DMDialect) batchInsertSimple(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	identityInsert := fmt.Sprintf("set identity_insert \"%s\" on;", tableName)

	sqlTemp := fmt.Sprintf("%s insert into %s (%s) values %s", identityInsert, dd.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)
	effRows := 0
	// 设置允许填充自增列之后，显示指定列名可以插入自增列
	for _, value := range values {
		// 达梦数据库只能一条条的执行insert
		res, err := dd.dc.TxExec(tx, sqlTemp, value...)
		if err != nil {
			logx.Errorf("执行sql失败：%s, sql: [ %s ]", err.Error(), sqlTemp)
			return 0, err
		}
		effRows += int(res)
	}
	// 执行批量insert sql
	return int64(effRows), nil
}

func (dd *DMDialect) batchInsertMerge(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	// 查询主键字段
	uniqueCols := make([]string, 0)
	caseSqls := make([]string, 0)
	metadata := dd.dc.GetMetadata()
	tableCols, _ := metadata.GetColumns(tableName)
	identityCols := make([]string, 0)
	for _, col := range tableCols {
		if col.IsPrimaryKey {
			uniqueCols = append(uniqueCols, col.ColumnName)
			caseSqls = append(caseSqls, fmt.Sprintf("( T1.%s = T2.%s )", dd.QuoteIdentifier(col.ColumnName), dd.QuoteIdentifier(col.ColumnName)))
		}
		if col.IsIdentity {
			// 自增字段不放入insert内，即使是设置了identity_insert on也不起作用
			identityCols = append(identityCols, dd.QuoteIdentifier(col.ColumnName))
		}
	}
	// 查询唯一索引涉及到的字段，并组装到match条件内
	indexs, _ := metadata.GetTableIndex(tableName)
	for _, index := range indexs {
		if index.IsUnique {
			cols := strings.Split(index.ColumnName, ",")
			tmp := make([]string, 0)
			for _, col := range cols {
				uniqueCols = append(uniqueCols, col)
				tmp = append(tmp, fmt.Sprintf(" T1.%s = T2.%s ", dd.QuoteIdentifier(col), dd.QuoteIdentifier(col)))
			}
			caseSqls = append(caseSqls, fmt.Sprintf("( %s )", strings.Join(tmp, " AND ")))
		}
	}

	// 重复数据处理策略
	phs := make([]string, 0)
	insertVals := make([]string, 0)
	upds := make([]string, 0)
	insertCols := make([]string, 0)
	for _, column := range columns {
		phs = append(phs, fmt.Sprintf("? %s", column))
		if !collx.ArrayContains(uniqueCols, dd.RemoveQuote(column)) {
			upds = append(upds, fmt.Sprintf("T1.%s = T2.%s", column, column))
		}
		if !collx.ArrayContains(identityCols, column) {
			insertCols = append(insertCols, column)
			insertVals = append(insertVals, fmt.Sprintf("T2.%s", column))
		}

	}
	t2s := make([]string, 0)
	for i := 0; i < len(values); i++ {
		t2s = append(t2s, fmt.Sprintf("SELECT %s FROM dual", strings.Join(phs, ",")))
	}
	t2 := strings.Join(t2s, " UNION ALL ")

	sqlTemp := "MERGE INTO " + dd.QuoteIdentifier(tableName) + " T1 USING (" + t2 + ") T2 ON " + strings.Join(caseSqls, " OR ")
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") VALUES (" + strings.Join(insertVals, ",") + ")"
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(upds, ",")

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}

	return dd.dc.TxExec(tx, sqlTemp, args...)
}

func (dd *DMDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName
	metadata := dd.dc.GetMetadata()
	ddl, err := metadata.GetTableDDL(tableName, false)
	if err != nil {
		return err
	}
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// 替换新表名
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("\"%s\"", strings.ToUpper(tableName)), fmt.Sprintf("\"%s\"", strings.ToUpper(newTableName)))
	// 去除空格换行
	ddl = stringx.TrimSpaceAndBr(ddl)
	// sqls, err := sqlparser.SplitStatementToPieces(ddl, sqlparser.WithDialect(dd.dc.GetMetaData().GetSqlParserDialect()))
	sqls := strings.Split(ddl, ";")
	for _, sql := range sqls {
		_, _ = dd.dc.Exec(sql)
	}

	// 复制数据
	if copy.CopyData {
		go func() {
			// 设置允许填充自增列之后，显示指定列名可以插入自增列\
			identityInsert := fmt.Sprintf("set identity_insert \"%s\" on", newTableName)
			// 获取列名
			columns, _ := metadata.GetColumns(tableName)
			columnArr := make([]string, 0)
			for _, column := range columns {
				columnArr = append(columnArr, fmt.Sprintf("\"%s\"", column.ColumnName))
			}
			columnStr := strings.Join(columnArr, ",")
			// 插入新数据并显示指定列
			_, _ = dd.dc.Exec(fmt.Sprintf("%s insert into \"%s\" (%s) select %s from \"%s\"", identityInsert, newTableName, columnStr, columnStr, tableName))

		}()
	}
	return err
}

func (dd *DMDialect) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	tbName := dd.QuoteIdentifier(tableInfo.TableName)
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("drop table if exists %s", tbName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("create table %s (", tbName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, dd.QuoteIdentifier(column.ColumnName))
		}
		fields = append(fields, dd.genColumnBasicSql(column))
		if column.ColumnComment != "" {
			comment := dd.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf("comment on column %s.%s is '%s'", tbName, dd.QuoteIdentifier(column.ColumnName), comment))
		}
	}
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(",\n PRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		comment := dd.QuoteEscape(tableInfo.TableComment)
		tableCommentSql = fmt.Sprintf("comment on table %s is '%s'", tbName, comment)
	}

	sqlArr = append(sqlArr, createSql)
	if tableCommentSql != "" {
		sqlArr = append(sqlArr, tableCommentSql)
	}

	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}

	return sqlArr
}

func (dd *DMDialect) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
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
			colNames[i] = dd.QuoteIdentifier(name)
		}

		sqls = append(sqls, fmt.Sprintf("create %s index %s on %s(%s)", unique, dd.QuoteIdentifier(index.IndexName), dd.QuoteIdentifier(tableInfo.TableName), strings.Join(colNames, ",")))
	}
	return sqls
}

func (dd *DMDialect) GetDataHelper() dbi.DataHelper {
	return new(DataHelper)
}

func (dd *DMDialect) GetColumnHelper() dbi.ColumnHelper {
	return new(ColumnHelper)
}

func (dd *DMDialect) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}

func (dd *DMDialect) genColumnBasicSql(column dbi.Column) string {
	colName := dd.QuoteIdentifier(column.ColumnName)
	dataType := string(column.DataType)

	incr := ""
	if column.IsIdentity {
		incr = " IDENTITY"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(dataType)) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) &&
				collx.ArrayAnyMatches([]string{"DATE", "TIME"}, strings.ToUpper(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
		}
		if mark {
			defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s", colName, column.GetColumnType(), incr, nullAble, defVal)
	return columnSql
}
