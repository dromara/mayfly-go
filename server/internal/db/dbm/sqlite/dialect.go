package sqlite

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

type SqliteDialect struct {
	dc *dbi.DbConn
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (sd *SqliteDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, fmt.Errorf("该数据库类型不支持数据库备份与恢复: %v", sd.dc.Info.Type)
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

	sqlStr := fmt.Sprintf("%s %s (%s) values %s", prefix, sd.dc.GetMetaData().QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)

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

func (sd *SqliteDialect) GetDataConverter() dbi.DataConverter {
	return converter
}

func (sd *SqliteDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")
	ddl, err := sd.dc.GetMetaData().GetTableDDL(tableName)
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

func (sd *SqliteDialect) TransColumns(columns []dbi.Column) []dbi.Column {
	var commonColumns []dbi.Column
	for _, column := range columns {
		// 取出当前数据库类型
		arr := strings.Split(column.ColumnType, "(")
		ctype := arr[0]
		// 翻译为通用数据库类型
		t1 := commonColumnTypeMap[ctype]
		if t1 == "" {
			ctype = "varchar(2000)"
		} else {
			// 回写到列信息
			if len(arr) > 1 {
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

func (sd *SqliteDialect) CreateTable(commonColumns []dbi.Column, tableInfo dbi.Table, dropOldTable bool) (int, error) {
	tbName := sd.dc.GetMetaData().QuoteIdentifier(tableInfo.TableName)
	if dropOldTable {
		_, err := sd.dc.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tbName))
		if err != nil {
			logx.Error("删除表失败", err)
		}
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", tbName)
	fields := make([]string, 0)
	// 把通用类型转换为达梦类型
	for _, column := range commonColumns {
		// 取出当前数据库类型
		arr := strings.Split(column.ColumnType, "(")
		ctype := arr[0]
		// 翻译为通用数据库类型
		t1 := sqliteColumnTypeMap[ctype]
		if t1 == "" {
			ctype = "nvarchar(2000)"
		} else {
			// 回写到列信息
			if len(arr) > 1 {
				ctype = t1 + "(" + arr[1]
			}
		}
		column.ColumnType = ctype
		fields = append(fields, sd.genColumnBasicSql(column))
	}
	createSql += strings.Join(fields, ",")
	createSql += fmt.Sprintf(") ")
	_, err := sd.dc.Exec(createSql)

	return 1, err
}

func (sd *SqliteDialect) CreateIndex(tableInfo dbi.Table, indexs []dbi.Index) error {
	sqls := make([]string, 0)
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
		sqlTmp := "CREATE %s INDEX %s ON \"%s\" (%s) "
		sqls = append(sqls, fmt.Sprintf(sqlTmp, unique, indexName, tableInfo.TableName, index.ColumnName))
	}
	_, err := sd.dc.Exec(strings.Join(sqls, ";"))
	return err
}

func (sd *SqliteDialect) genColumnBasicSql(column dbi.Column) string {

	incr := ""
	if column.IsIdentity {
		incr = " AUTOINCREMENT"
	}

	nullAble := ""
	if column.Nullable == "NO" {
		nullAble = " NOT NULL"
	}

	// 如果是主键，则直接返回，不判断默认值
	if column.IsPrimaryKey {
		return fmt.Sprintf(" %s integer PRIMARY KEY %s %s", column.ColumnName, incr, nullAble)
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(column.ColumnType)) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(column.ColumnType)) &&
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

	return fmt.Sprintf(" %s %s %s %s", sd.dc.GetMetaData().QuoteIdentifier(column.ColumnName), column.ColumnType, nullAble, defVal)
}

func (sd *SqliteDialect) UpdateSequence(tableName string, columns []dbi.Column) {

}
