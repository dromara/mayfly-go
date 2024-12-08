package sqlite

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

type SqliteDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (sd *SqliteDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
	_, _ = sd.dc.Exec("PRAGMA foreign_keys = false")
	// 执行批量insert sql，跟mysql一样 支持批量insert语法
	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	// 重复占位符字符串n遍
	repeated = strings.Repeat(placeholder+",", len(values))
	// 去除最后一个逗号
	placeholder = strings.TrimSuffix(repeated, ",")

	prefix := "insert into"
	if duplicateStrategy == 1 {
		prefix = "insert or ignore into"
	} else if duplicateStrategy == 2 {
		prefix = "insert or replace into"
	}

	sqlStr := fmt.Sprintf("%s %s (%s) values %s", prefix, sd.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}
	exec, err := sd.dc.TxExec(tx, sqlStr, args...)

	_, _ = sd.dc.Exec("PRAGMA foreign_keys = true;")

	// 执行批量insert sql
	return exec, err
}

func (sd *SqliteDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")
	ddl, err := sd.dc.GetMetadata().GetTableDDL(tableName, false)
	if err != nil {
		return err
	}
	// 生成建表语句
	// 替换表名
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("CREATE TABLE \"%s\"", tableName), fmt.Sprintf("CREATE TABLE \"%s\"", newTableName))
	// 替换索引名，索引名为按照规范生成的，才能替换，否则未知索引名，无法替换
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("CREATE INDEX \"%s", tableName), fmt.Sprintf("CREATE INDEX \"%s", newTableName))

	// 执行建表语句
	_, err = sd.dc.Exec(ddl)
	if err != nil {
		return err
	}

	// 使用异步线程插入数据
	if copy.CopyData {
		go func() {
			// 执行插入语句
			_, _ = sd.dc.Exec(fmt.Sprintf("INSERT INTO \"%s\" SELECT * FROM \"%s\"", newTableName, tableName))
		}()
	}

	return err
}

// 获取建表ddl
func (sd *SqliteDialect) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	sqlArr := make([]string, 0)
	tbName := sd.QuoteIdentifier(tableInfo.TableName)
	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", tbName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", tbName)
	fields := make([]string, 0)

	// 把通用类型转换为达梦类型
	for _, column := range columns {
		fields = append(fields, sd.genColumnBasicSql(column))
	}
	createSql += strings.Join(fields, ",\n")
	createSql += "\n)"

	sqlArr = append(sqlArr, createSql)

	return sqlArr
}

func (sd *SqliteDialect) genColumnBasicSql(column dbi.Column) string {
	incr := ""
	if column.IsIdentity {
		incr = " AUTOINCREMENT"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	quoteColumnName := sd.QuoteIdentifier(column.ColumnName)

	// 如果是主键，则直接返回，不判断默认值
	if column.IsPrimaryKey {
		return fmt.Sprintf(" %s integer PRIMARY KEY%s%s", quoteColumnName, incr, nullAble)
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(string(column.DataType))) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(string(column.DataType))) &&
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

	return fmt.Sprintf(" %s %s%s%s", quoteColumnName, column.GetColumnType(), nullAble, defVal)
}

// 获取建索引ddl
func (sd *SqliteDialect) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
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
			colNames[i] = sd.QuoteIdentifier(name)
		}
		// 创建前尝试删除
		sqls = append(sqls, fmt.Sprintf("DROP INDEX IF EXISTS \"%s\"", index.IndexName))

		sqlTmp := "CREATE %s INDEX %s ON %s (%s) "
		sqls = append(sqls, fmt.Sprintf(sqlTmp, unique, sd.QuoteIdentifier(index.IndexName), sd.QuoteIdentifier(tableInfo.TableName), strings.Join(colNames, ",")))
	}
	return sqls
}

func (sd *SqliteDialect) GetDataHelper() dbi.DataHelper {
	return new(DataHelper)
}

func (sd *SqliteDialect) GetColumnHelper() dbi.ColumnHelper {
	return new(ColumnHelper)
}

func (sd *SqliteDialect) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}
