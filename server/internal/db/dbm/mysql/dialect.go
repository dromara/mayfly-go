package mysql

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

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

	sqlStr := fmt.Sprintf("%s %s (%s) values %s", prefix, md.dc.GetMetaData().QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)
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

func (md *MysqlDialect) ToCommonColumn(column *dbi.Column) {
	dataType := column.DataType

	t1 := commonColumnTypeMap[string(dataType)]
	commonColumnType := dbi.CommonTypeVarchar

	if t1 != "" {
		commonColumnType = t1
	}

	column.DataType = commonColumnType
}

func (md *MysqlDialect) ToColumn(column *dbi.Column) {
	ctype := mysqlColumnTypeMap[column.DataType]
	if ctype == "" {
		column.DataType = "varchar"
		column.CharMaxLength = 1000
	} else {
		column.DataType = dbi.ColumnDataType(ctype)
	}
}

func (md *MysqlDialect) CreateTable(columns []dbi.Column, tableInfo dbi.Table, dropOldTable bool) (int, error) {
	if dropOldTable {
		_, _ = md.dc.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableInfo.TableName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", tableInfo.TableName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	// 把通用类型转换为达梦类型
	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, column.ColumnName)
		}
		fields = append(fields, md.genColumnBasicSql(column))
	}
	createSql += strings.Join(fields, ",")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 "
	if tableInfo.TableComment != "" {
		replacer := strings.NewReplacer(";", "", "'", "")
		createSql += fmt.Sprintf(" COMMENT '%s'", replacer.Replace(tableInfo.TableComment))
	}
	_, err := md.dc.Exec(createSql)

	return 1, err
}

func (md *MysqlDialect) genColumnBasicSql(column dbi.Column) string {

	incr := ""
	if column.IsIdentity {
		incr = " AUTO_INCREMENT"
	}

	nullAble := ""
	if column.Nullable == "NO" {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的  // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
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
	comment := ""
	if column.ColumnComment != "" {
		// 防止注释内含有特殊字符串导致sql出错
		replacer := strings.NewReplacer(";", "", "'", "")
		commentStr := replacer.Replace(column.ColumnComment)
		comment = fmt.Sprintf(" COMMENT '%s'", commentStr)
	}

	columnSql := fmt.Sprintf(" %s %s %s %s %s %s", md.dc.GetMetaData().QuoteIdentifier(column.ColumnName), column.GetColumnType(), nullAble, incr, defVal, comment)
	return columnSql
}

func (md *MysqlDialect) CreateIndex(tableInfo dbi.Table, indexs []dbi.Index) error {
	meta := md.dc.GetMetaData()
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
		sqlTmp := "ALTER TABLE %s ADD %s INDEX %s(%s) USING BTREE COMMENT '%s'"
		replacer := strings.NewReplacer(";", "", "'", "")
		_, err := md.dc.Exec(fmt.Sprintf(sqlTmp, meta.QuoteIdentifier(tableInfo.TableName), unique, indexName, index.ColumnName, replacer.Replace(index.IndexComment)))
		if err != nil {
			return err
		}
	}
	return nil
}
