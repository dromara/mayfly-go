package mysql

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
	"regexp"
	"strings"
	"time"
)

type MysqlDialect struct {
	dc *dbi.DbConn
}

func (md *MysqlDialect) GetMetaData() dbi.MetaData {
	return &MysqlMetaData{dc: md.dc}
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

	sqlStr := fmt.Sprintf("%s %s (%s) values %s", prefix, md.dc.Info.Type.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)
	// 执行批量insert sql
	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}
	return md.dc.TxExec(tx, sqlStr, args...)
}

func (md *MysqlDialect) GetDataConverter() dbi.DataConverter {
	return converter
}

var (
	// 数字类型
	numberRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`)
	// 日期时间类型
	datetimeRegexp = regexp.MustCompile(`(?i)datetime|timestamp`)
	// 日期类型
	dateRegexp = regexp.MustCompile(`(?i)date`)
	// 时间类型
	timeRegexp = regexp.MustCompile(`(?i)time`)

	converter = new(DataConverter)
)

type DataConverter struct {
}

func (dc *DataConverter) GetDataType(dbColumnType string) dbi.DataType {
	if numberRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	// 日期时间类型
	if datetimeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	// 日期类型
	if dateRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDate
	}
	// 时间类型
	if timeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeTime
	}
	return dbi.DataTypeString
}

func (dc *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
	return anyx.ToString(dbColumnValue)
}

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	// 如果dataType是datetime而dbColumnValue是string类型，则需要转换为time.Time类型
	_, ok := dbColumnValue.(string)
	if ok {
		if dataType == dbi.DataTypeDateTime {
			res, _ := time.Parse(time.DateTime, anyx.ToString(dbColumnValue))
			return res
		}
		if dataType == dbi.DataTypeDate {
			res, _ := time.Parse(time.DateOnly, anyx.ToString(dbColumnValue))
			return res
		}
		if dataType == dbi.DataTypeTime {
			res, _ := time.Parse(time.TimeOnly, anyx.ToString(dbColumnValue))
			return res
		}
	}
	return dbColumnValue
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
