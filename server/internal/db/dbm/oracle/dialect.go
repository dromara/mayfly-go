package oracle

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"

	_ "gitee.com/chunanyong/dm"
)

type OracleDialect struct {
	dc *dbi.DbConn
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (od *OracleDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, fmt.Errorf("该数据库类型不支持数据库备份与恢复: %v", od.dc.Info.Type)
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
	metadata := od.dc.GetMetaData()
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
		sqlTemp := fmt.Sprintf("INSERT %s INTO %s (%s) VALUES (%s)", ignore, metadata.QuoteIdentifier(tableName), strings.Join(columns, ","), strings.Join(placeholder, ","))

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
	metadata := od.dc.GetMetaData()
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
					tmp = append(tmp, fmt.Sprintf(" T1.%s = T2.%s ", metadata.QuoteIdentifier(col), metadata.QuoteIdentifier(col)))
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
		if !collx.ArrayContains(uniqueCols, metadata.RemoveQuote(column)) {
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

	sqlTemp := "MERGE INTO " + metadata.QuoteIdentifier(tableName) + " T1 USING (" + t2 + ") T2 ON (" + strings.Join(caseSqls, " OR ") + ") "
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

func (od *OracleDialect) TransColumns(columns []dbi.Column) []dbi.Column {
	var commonColumns []dbi.Column
	for _, column := range columns {
		// 取出当前数据库类型
		arr := strings.Split(column.ColumnType, "(")
		ctype := arr[0]

		// 翻译为通用数据库类型
		t1 := commonColumnTypeMap[ctype]
		if t1 == "" {
			ctype = "NVARCHAR2(2000)"
		} else {
			// 回写到列信息
			if t1 == "NUMBER" {
				// 如果是转number类型，需要根据公共类型加上长度, 如 bigint 需要转换为number(19,0)
				if column.ColumnType == dbi.CommonTypeBigint {
					ctype = t1 + "(19, 0)"
				} else {
					ctype = t1
				}
			} else if t1 != "NUMBER" && len(arr) > 1 {
				ctype = t1 + "(" + arr[1]
			} else {
				ctype = t1
			}
		}
		column.ColumnType = ctype
		commonColumns = append(commonColumns, column)
	}
	return commonColumns
}

func (od *OracleDialect) CreateTable(commonColumns []dbi.Column, tableInfo dbi.Table, dropOldTable bool) (int, error) {
	meta := od.dc.GetMetaData()
	replacer := strings.NewReplacer(";", "", "'", "")
	quoteTableName := meta.QuoteIdentifier(tableInfo.TableName)
	if dropOldTable {
		// 如果表存在，先删除表
		dropSqlTmp := `
declare
      num number;
begin
    select count(1) into num from user_tables where table_name = '%s' and owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual) ;
    if num > 0 then
        execute immediate 'drop table "%s"' ;
    end if;
end;
`
		_, _ = od.dc.Exec(fmt.Sprintf(dropSqlTmp, tableInfo.TableName, tableInfo.TableName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (", quoteTableName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)
	// 把通用类型转换为达梦类型
	for _, column := range commonColumns {
		// 取出当前数据库类型
		arr := strings.Split(column.ColumnType, "(")
		ctype := arr[0]
		// 翻译为通用数据库类型
		t1 := oracleColumnTypeMap[ctype]
		if t1 == "" {
			ctype = "NVARCHAR2(2000)"
		} else {
			// 回写到列信息
			if len(arr) > 1 {
				// 如果是字符串类型，长度最大4000，否则修改字段类型为clob
				if strings.Contains(strings.ToLower(t1), "char") {
					match := bracketsRegexp.FindStringSubmatch(column.ColumnType)
					if len(match) > 1 {
						size := anyx.ConvInt(match[1])
						if size >= 4000 { // 如果长度超过4000，则替换为text类型
							ctype = "CLOB"
						} else {
							ctype = fmt.Sprintf("%s(%d)", t1, size)
						}
					} else {
						ctype = t1 + "(2000)"
					}
				} else {
					ctype = t1 + "(" + arr[1]
				}
			} else {
				ctype = t1
			}
		}
		column.ColumnType = ctype

		if column.IsPrimaryKey {
			pks = append(pks, meta.QuoteIdentifier(column.ColumnName))
		}
		fields = append(fields, od.genColumnBasicSql(column))
		// 防止注释内含有特殊字符串导致sql出错
		comment := replacer.Replace(column.ColumnComment)
		if comment != "" {
			columnComments = append(columnComments, fmt.Sprintf("COMMENT ON COLUMN %s.%s IS '%s'", quoteTableName, meta.QuoteIdentifier(column.ColumnName), comment))
		}
	}
	createSql += strings.Join(fields, ",")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += ")"

	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		tableCommentSql = fmt.Sprintf(" COMMENT ON TABLE %s is '%s'", meta.QuoteIdentifier(tableInfo.TableName), replacer.Replace(tableInfo.TableComment))
	}

	// 需要分开执行sql
	var err error
	if createSql != "" {
		_, err = od.dc.Exec(createSql)
	}
	if tableCommentSql != "" {
		_, err = od.dc.Exec(tableCommentSql)
	}
	if len(columnComments) > 0 {
		for _, commentSql := range columnComments {
			_, err = od.dc.Exec(commentSql)
		}
	}

	return 1, err
}

func (od *OracleDialect) genColumnBasicSql(column dbi.Column) string {
	meta := od.dc.GetMetaData()
	colName := meta.QuoteIdentifier(column.ColumnName)

	if column.IsIdentity {
		// 如果是自增，不需要设置默认值和空值，自增列数据类型必须是number
		return fmt.Sprintf(" %s NUMBER generated by default as IDENTITY", colName)
	}

	nullAble := ""
	if column.Nullable == "NO" {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的
	if column.ColumnDefault != "" {
		mark := false
		// 哪些字段类型默认值需要加引号
		if collx.ArrayAnyMatches([]string{"CHAR", "LONG", "DATE", "TIME", "CLOB", "BLOB", "BFILE"}, column.ColumnType) {
			// 默认值是时间日期函数的必须要加引号
			val := strings.ToUpper(column.ColumnDefault)
			if collx.ArrayAnyMatches([]string{"DATE", "TIMESTAMP"}, column.ColumnType) && val == "CURRENT_DATE" || val == "CURRENT_TIMESTAMP" {
				mark = false
			} else {
				mark = true
			}
			if mark {
				defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
			} else {
				defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
			}
		} else {
			// 如果是数字，默认值提取数字
			if collx.ArrayAnyMatches([]string{"NUM", "INT"}, column.ColumnType) {
				match := bracketsRegexp.FindStringSubmatch(column.ColumnType)
				if len(match) > 1 {
					length := anyx.ConvInt(match[1])
					defVal = fmt.Sprintf(" DEFAULT %d", length)
				} else {
					defVal = fmt.Sprintf(" DEFAULT 0")
				}
			}

			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	columnSql := fmt.Sprintf(" %s %s %s %s", colName, column.ColumnType, defVal, nullAble)
	return columnSql
}

func (od *OracleDialect) CreateIndex(tableInfo dbi.Table, indexs []dbi.Index) error {
	meta := od.dc.GetMetaData()
	sqls := make([]string, 0)
	comments := make([]string, 0)
	for _, index := range indexs {
		// 通过字段、表名拼接索引名
		columnName := strings.ReplaceAll(index.ColumnName, "-", "")
		columnName = strings.ReplaceAll(columnName, "_", "")
		colName := strings.ReplaceAll(columnName, ",", "_")

		keyType := "normal"
		unique := ""
		if index.IsUnique {
			keyType = "unique"
			unique = "unique"
		}
		indexName := fmt.Sprintf("%s_key_%s_%s", keyType, tableInfo.TableName, colName)

		sqls = append(sqls, fmt.Sprintf("CREATE %s INDEX %s ON %s(%s)", unique, indexName, meta.QuoteIdentifier(tableInfo.TableName), index.ColumnName))
		if index.IndexComment != "" {
			comments = append(comments, fmt.Sprintf("COMMENT ON INDEX %s IS '%s'", indexName, index.IndexComment))
		}
	}
	_, err := od.dc.Exec(strings.Join(sqls, ";"))

	// 添加注释
	if len(comments) > 0 {
		_, err = od.dc.Exec(strings.Join(comments, ";"))
	}
	return err
}

func (od *OracleDialect) UpdateSequence(tableName string, columns []dbi.Column) {

}
