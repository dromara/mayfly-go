package oracle

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"

	_ "gitee.com/chunanyong/dm"
)

type OracleDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (od *OracleDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
	if len(values) <= 0 {
		return 0, nil
	}

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}

	if duplicateStrategy == dbi.DuplicateStrategyNone || duplicateStrategy == 0 || duplicateStrategy == dbi.DuplicateStrategyIgnore {
		return od.batchInsertSimple(tableName, columns, values, duplicateStrategy, tx)
	} else {
		return od.batchInsertMergeSql(tableName, columns, values, args, tx)
	}
}

// 简单批量插入sql，无需判断键冲突策略
func (od *OracleDialect) batchInsertSimple(tableName string, columns []string, values [][]any, duplicateStrategy int, tx *sql.Tx) (int64, error) {
	metadata := od.dc.GetMetadata()
	// 忽略键冲突策略
	ignore := ""
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		// 查出唯一索引涉及的字段
		indexs, _ := metadata.GetTableIndex(tableName)
		if indexs != nil {
			arr := make([]string, 0)
			for _, index := range indexs {
				if index.IsUnique {
					cols := strings.Split(index.ColumnName, ",")
					for _, col := range cols {
						if !collx.ArrayContains(arr, col) {
							arr = append(arr, col)
						}
					}
				}
			}
			ignore = fmt.Sprintf("/*+ IGNORE_ROW_ON_DUPKEY_INDEX(%s(%s)) */", tableName, strings.Join(arr, ","))
		}
	}
	effRows := 0
	for _, value := range values {
		// 拼接带占位符的sql oracle的占位符是:1,:2,:3....
		var placeholder []string
		for i := 0; i < len(value); i++ {
			placeholder = append(placeholder, fmt.Sprintf(":%d", i+1))
		}
		sqlTemp := fmt.Sprintf("INSERT %s INTO %s (%s) VALUES (%s)", ignore, od.QuoteIdentifier(tableName), strings.Join(columns, ","), strings.Join(placeholder, ","))

		// oracle数据库为了兼容ignore主键冲突，只能一条条的执行insert
		res, err := od.dc.TxExec(tx, sqlTemp, value...)
		if err != nil {
			logx.Errorf("执行sql失败：%s, sql: [ %s ]", err.Error(), sqlTemp)
		}
		effRows += int(res)
	}
	return int64(effRows), nil
}

func (od *OracleDialect) batchInsertMergeSql(tableName string, columns []string, values [][]any, args []any, tx *sql.Tx) (int64, error) {
	// 查询主键字段
	uniqueCols := make([]string, 0)
	caseSqls := make([]string, 0)
	metadata := od.dc.GetMetadata()
	// 查询唯一索引涉及到的字段，并组装到match条件内
	indexs, _ := metadata.GetTableIndex(tableName)
	if indexs != nil {
		for _, index := range indexs {
			if index.IsUnique {
				cols := strings.Split(index.ColumnName, ",")
				tmp := make([]string, 0)
				for _, col := range cols {
					if !collx.ArrayContains(uniqueCols, col) {
						uniqueCols = append(uniqueCols, col)
					}
					tmp = append(tmp, fmt.Sprintf(" T1.%s = T2.%s ", od.QuoteIdentifier(col), od.QuoteIdentifier(col)))
				}
				caseSqls = append(caseSqls, fmt.Sprintf("( %s )", strings.Join(tmp, " AND ")))
			}
		}
	}

	// 如果caseSqls为空，则说明没有唯一键，直接使用简单批量插入
	if len(caseSqls) == 0 {
		return od.batchInsertSimple(tableName, columns, values, dbi.DuplicateStrategyNone, tx)
	}

	// 重复数据处理策略
	insertVals := make([]string, 0)
	upds := make([]string, 0)
	insertCols := make([]string, 0)
	for _, column := range columns {
		if !collx.ArrayContains(uniqueCols, od.RemoveQuote(column)) {
			upds = append(upds, fmt.Sprintf("T1.%s = T2.%s", column, column))
		}
		insertCols = append(insertCols, fmt.Sprintf("T1.%s", column))
		insertVals = append(insertVals, fmt.Sprintf("T2.%s", column))
	}

	// 生成源数据占位sql
	t2s := make([]string, 0)
	// 拼接带占位符的sql oracle的占位符是:1,:2,:3....
	for i := 0; i < len(args); i += len(columns) {
		var placeholder []string
		for j := 0; j < len(columns); j++ {
			col := columns[j]
			placeholder = append(placeholder, fmt.Sprintf(":%d %s", i+j+1, col))
		}
		t2s = append(t2s, fmt.Sprintf("SELECT %s FROM dual", strings.Join(placeholder, ", ")))
	}

	t2 := strings.Join(t2s, " UNION ALL ")

	sqlTemp := "MERGE INTO " + od.QuoteIdentifier(tableName) + " T1 USING (" + t2 + ") T2 ON (" + strings.Join(caseSqls, " OR ") + ") "
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") VALUES (" + strings.Join(insertVals, ",") + ") "
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(upds, ",")

	// 执行批量insert sql
	res, err := od.dc.TxExec(tx, sqlTemp, args...)
	if err != nil {
		logx.Errorf("执行sql失败：%s, sql: [ %s ]", err.Error(), sqlTemp)
	}
	return res, err
}

func (od *OracleDialect) CopyTable(copy *dbi.DbCopyTable) error {
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := strings.ToUpper(copy.TableName + "_copy_" + time.Now().Format("20060102150405"))
	condition := ""
	if !copy.CopyData {
		condition = " where 1 = 2"
	}
	_, err := od.dc.Exec(fmt.Sprintf("create table \"%s\" as select * from \"%s\" %s", newTableName, copy.TableName, condition))
	return err
}

// 获取建表ddl
func (od *OracleDialect) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	quoteTableName := od.QuoteIdentifier(tableInfo.TableName)
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		dropSqlTmp := `
declare
      num number;
begin
    select count(1) into num from user_tables where table_name = '%s' and owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual) ;
    if num > 0 then
        execute immediate 'drop table "%s"' ;
    end if;
end`
		sqlArr = append(sqlArr, fmt.Sprintf(dropSqlTmp, tableInfo.TableName, tableInfo.TableName))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s ( \n", quoteTableName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)
	// 把通用类型转换为达梦类型
	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, od.QuoteIdentifier(column.ColumnName))
		}
		fields = append(fields, od.genColumnBasicSql(column))
		// 防止注释内含有特殊字符串导致sql出错
		if column.ColumnComment != "" {
			comment := od.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf("COMMENT ON COLUMN %s.%s IS '%s'", quoteTableName, od.QuoteIdentifier(column.ColumnName), comment))
		}
	}

	// 建表
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"
	sqlArr = append(sqlArr, createSql)

	// 表注释
	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		tableCommentSql = fmt.Sprintf("COMMENT ON TABLE %s is '%s'", od.QuoteIdentifier(tableInfo.TableName), od.QuoteEscape(tableInfo.TableComment))
		sqlArr = append(sqlArr, tableCommentSql)
	}

	// 列注释
	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}
	otherSql := od.GenerateTableOtherDDL(tableInfo, quoteTableName, columns)
	if len(otherSql) > 0 {
		sqlArr = append(sqlArr, otherSql...)
	}

	return sqlArr
}

// 获取建索引ddl
func (od *OracleDialect) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
	sqls := make([]string, 0)
	comments := make([]string, 0)

	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}

		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, name := range cols {
			colNames[i] = od.QuoteIdentifier(name)
		}

		sqls = append(sqls, fmt.Sprintf("CREATE %s INDEX %s ON %s(%s)", unique, od.QuoteIdentifier(index.IndexName), od.QuoteIdentifier(tableInfo.TableName), strings.Join(colNames, ",")))
	}

	sqlArr := make([]string, 0)

	sqlArr = append(sqlArr, sqls...)

	if len(comments) > 0 {
		sqlArr = append(sqlArr, comments...)
	}

	return sqlArr
}

// 11g及以下版本会设置自增序列
func (od *OracleDialect) GenerateTableOtherDDL(tableInfo dbi.Table, quoteTableName string, columns []dbi.Column) []string {
	return nil
}

func (od *OracleDialect) GetDataHelper() dbi.DataHelper {
	return new(DataHelper)
}

func (od *OracleDialect) GetColumnHelper() dbi.ColumnHelper {
	return new(ColumnHelper)
}

func (od *OracleDialect) genColumnBasicSql(column dbi.Column) string {
	colName := od.QuoteIdentifier(column.ColumnName)

	if column.IsIdentity {
		// 如果是自增，不需要设置默认值和空值，自增列数据类型必须是number
		return fmt.Sprintf(" %s NUMBER generated by default as IDENTITY", colName)
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := ""
	if column.ColumnDefault != "" {
		defVal = fmt.Sprintf(" DEFAULT %v", column.ColumnDefault)
	}

	columnSql := fmt.Sprintf(" %s %s%s%s", colName, column.GetColumnType(), defVal, nullAble)
	return columnSql
}
