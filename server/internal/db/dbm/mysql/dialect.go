package mysql

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

const Quoter = "`"

type MysqlDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (md *MysqlDialect) GetDbProgram() (dbi.DbProgram, error) {
	return NewDbProgramMysql(md.dc), nil
}

func (md *MysqlDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	// 执行批量insert sql，mysql支持批量insert语法
	// insert into table_name (column1, column2, ...) values (value1, value2, ...), (value1, value2, ...), ...

	// 重复占位符字符串n遍
	repeated = strings.Repeat(placeholder+",", len(values))
	// 去除最后一个逗号
	placeholder = strings.TrimSuffix(repeated, ",")

	prefix := "insert into"
	if duplicateStrategy == 1 {
		prefix = "insert ignore into"
	} else if duplicateStrategy == 2 {
		prefix = "replace into"
	}

	sqlStr := fmt.Sprintf("%s %s (%s) values %s", prefix, md.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)
	// 执行批量insert sql
	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}
	return md.dc.TxExec(tx, sqlStr, args...)
}

func (md *MysqlDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// 复制表结构创建表
	_, err := md.dc.Exec(fmt.Sprintf("create table %s like %s", newTableName, tableName))
	if err != nil {
		return err
	}

	// 复制数据
	if copy.CopyData {
		go func() {
			_, _ = md.dc.Exec(fmt.Sprintf("insert into %s select * from %s", newTableName, tableName))
		}()
	}
	return err
}

// 获取建表ddl
func (md *MysqlDialect) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", md.QuoteIdentifier(tableInfo.TableName)))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", md.QuoteIdentifier(tableInfo.TableName))
	fields := make([]string, 0)
	pks := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, column.ColumnName)
		}
		fields = append(fields, md.genColumnBasicSql(column))
	}

	// 建表ddl
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	// 表注释
	if tableInfo.TableComment != "" {
		createSql += fmt.Sprintf(" COMMENT '%s'", md.QuoteEscape(tableInfo.TableComment))
	}

	sqlArr = append(sqlArr, createSql)

	return sqlArr
}

// 获取建索引ddl
func (md *MysqlDialect) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
	sqlArr := make([]string, 0)
	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}
		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, name := range cols {
			colNames[i] = md.QuoteIdentifier(name)
		}
		sqlTmp := "ALTER TABLE %s ADD %s INDEX %s(%s) USING BTREE"
		sqlStr := fmt.Sprintf(sqlTmp, md.QuoteIdentifier(tableInfo.TableName), unique, md.QuoteIdentifier(index.IndexName), strings.Join(colNames, ","))
		comment := md.QuoteEscape(index.IndexComment)
		if comment != "" {
			sqlStr += fmt.Sprintf(" COMMENT '%s'", comment)
		}
		sqlArr = append(sqlArr, sqlStr)
	}
	return sqlArr
}

func (md *MysqlDialect) genColumnBasicSql(column dbi.Column) string {
	dataType := string(column.DataType)

	incr := ""
	if column.IsIdentity {
		incr = " AUTO_INCREMENT"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}
	columnType := column.GetColumnType()
	if nullAble == "" && strings.Contains(columnType, "timestamp") {
		nullAble = " NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的
	if column.ColumnDefault != "" &&
		// 当默认值是字符串'NULL'时，不需要设置默认值
		column.ColumnDefault != "NULL" &&
		// 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
		!strings.Contains(column.ColumnDefault, "(") {
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
	comment := ""
	if column.ColumnComment != "" {
		// 防止注释内含有特殊字符串导致sql出错
		commentStr := md.QuoteEscape(column.ColumnComment)
		comment = fmt.Sprintf(" COMMENT '%s'", commentStr)
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s%s", md.QuoteIdentifier(column.ColumnName), columnType, nullAble, incr, defVal, comment)
	return columnSql
}

func (dx *MysqlDialect) QuoteIdentifier(name string) string {
	end := strings.IndexRune(name, 0)
	if end > -1 {
		name = name[:end]
	}
	return Quoter + strings.Replace(name, Quoter, Quoter+Quoter, -1) + Quoter
}

func (dx *MysqlDialect) RemoveQuote(name string) string {
	return strings.ReplaceAll(name, Quoter, "")
}

func (md *MysqlDialect) QuoteLiteral(literal string) string {
	literal = strings.ReplaceAll(literal, `\`, `\\`)
	literal = strings.ReplaceAll(literal, `'`, `''`)
	return "'" + literal + "'"
}

func (md *MysqlDialect) GetDataHelper() dbi.DataHelper {
	return dataHelper
}

func (md *MysqlDialect) GetColumnHelper() dbi.ColumnHelper {
	return columnHelper
}

func (pd *MysqlDialect) GetSQLParser() sqlparser.SqlParser {
	return new(mysql.MysqlParser)
}
