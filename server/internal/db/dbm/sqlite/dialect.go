package sqlite

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
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

func (sd *SqliteDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")
	ddl, err := sd.dc.GetMetaData().GetTableDDL(tableName, false)
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

func (sd *SqliteDialect) CreateTable(columns []dbi.Column, tableInfo dbi.Table, dropOldTable bool) (int, error) {
	sqlArr := sd.dc.GetMetaData().GenerateTableDDL(columns, tableInfo, dropOldTable)
	for _, sqlStr := range sqlArr {
		_, err := sd.dc.Exec(sqlStr)
		if err != nil {
			return 0, err
		}
	}
	return len(sqlArr), nil
}

func (sd *SqliteDialect) CreateIndex(tableInfo dbi.Table, indexs []dbi.Index) error {
	sqlArr := sd.dc.GetMetaData().GenerateIndexDDL(indexs, tableInfo)
	for _, sqlStr := range sqlArr {
		_, err := sd.dc.Exec(sqlStr)
		if err != nil {
			return err
		}
	}
	return nil
}
