package sqlite

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
	"regexp"
	"strings"
	"time"
)

type SqliteDialect struct {
	dc *dbi.DbConn
}

func (sd *SqliteDialect) GetMetaData() dbi.MetaData {
	return &SqliteMetaData{dc: sd.dc}
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (sd *SqliteDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, fmt.Errorf("该数据库类型不支持数据库备份与恢复: %v", sd.dc.Info.Type)
}

func (sd *SqliteDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
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

	sqlStr := fmt.Sprintf("%s %s (%s) values %s", prefix, sd.dc.Info.Type.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}

	// 执行批量insert sql
	return sd.dc.TxExec(tx, sqlStr, args...)
}

func (sd *SqliteDialect) GetDataConverter() dbi.DataConverter {
	return converter
}

var (
	// 数字类型
	numberRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit|real`)
	// 日期时间类型
	datetimeRegexp = regexp.MustCompile(`(?i)datetime`)

	converter = new(DataConverter)
)

type DataConverter struct {
}

func (dc *DataConverter) GetDataType(dbColumnType string) dbi.DataType {
	if numberRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	if datetimeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	return dbi.DataTypeString
}

func (dc *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
	str := anyx.ToString(dbColumnValue)
	switch dataType {
	case dbi.DataTypeDateTime: // "2024-01-02T22:08:22.275697+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.DateTime)
	case dbi.DataTypeDate: // "2024-01-02T00:00:00+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.DateOnly)
	case dbi.DataTypeTime: // "0000-01-01T22:08:22.275688+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.TimeOnly)
	}
	return str
}

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	return dbColumnValue
}

func (sd *SqliteDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")
	ddl, err := sd.GetMetaData().GetTableDDL(tableName)
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
