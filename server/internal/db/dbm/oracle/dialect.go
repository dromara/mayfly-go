package oracle

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"
	"time"

	_ "gitee.com/chunanyong/dm"
)

type OracleDialect struct {
	dc *dbi.DbConn
}

func (od *OracleDialect) GetMetaData() dbi.MetaData {
	return &OracleMetaData{dc: od.dc}
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
		return od.batchInsertSimple(od.dc.Info.Type, tableName, columns, values, duplicateStrategy, tx)
	} else {
		return od.batchInsertMergeSql(od.dc.Info.Type, tableName, columns, values, args, tx)
	}
}

// 简单批量插入sql，无需判断键冲突策略
func (od *OracleDialect) batchInsertSimple(dbType dbi.DbType, tableName string, columns []string, values [][]any, duplicateStrategy int, tx *sql.Tx) (int64, error) {

	// 忽略键冲突策略
	ignore := ""
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		// 查出唯一索引涉及的字段
		indexs, _ := od.GetMetaData().GetTableIndex(tableName)
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
		sqlTemp := fmt.Sprintf("INSERT %s INTO %s (%s) VALUES (%s)", ignore, dbType.QuoteIdentifier(tableName), strings.Join(columns, ","), strings.Join(placeholder, ","))

		// oracle数据库为了兼容ignore主键冲突，只能一条条的执行insert
		res, err := od.dc.TxExec(tx, sqlTemp, value...)
		if err != nil {
			logx.Errorf("执行sql失败：%s, sql: [ %s ]", err.Error(), sqlTemp)
		}
		effRows += int(res)
	}
	return int64(effRows), nil
}

func (od *OracleDialect) batchInsertMergeSql(dbType dbi.DbType, tableName string, columns []string, values [][]any, args []any, tx *sql.Tx) (int64, error) {
	// 查询主键字段
	uniqueCols := make([]string, 0)
	caseSqls := make([]string, 0)
	// 查询唯一索引涉及到的字段，并组装到match条件内
	indexs, _ := od.GetMetaData().GetTableIndex(tableName)
	if indexs != nil {
		for _, index := range indexs {
			if index.IsUnique {
				cols := strings.Split(index.ColumnName, ",")
				tmp := make([]string, 0)
				for _, col := range cols {
					if !collx.ArrayContains(uniqueCols, col) {
						uniqueCols = append(uniqueCols, col)
					}
					tmp = append(tmp, fmt.Sprintf(" T1.%s = T2.%s ", dbType.QuoteIdentifier(col), dbType.QuoteIdentifier(col)))
				}
				caseSqls = append(caseSqls, fmt.Sprintf("( %s )", strings.Join(tmp, " AND ")))
			}
		}
	}

	// 如果caseSqls为空，则说明没有唯一键，直接使用简单批量插入
	if len(caseSqls) == 0 {
		return od.batchInsertSimple(dbType, tableName, columns, values, dbi.DuplicateStrategyNone, tx)
	}

	// 重复数据处理策略
	insertVals := make([]string, 0)
	upds := make([]string, 0)
	insertCols := make([]string, 0)
	for _, column := range columns {
		if !collx.ArrayContains(uniqueCols, dbType.RemoveQuote(column)) {
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

	sqlTemp := "MERGE INTO " + dbType.QuoteIdentifier(tableName) + " T1 USING (" + t2 + ") T2 ON (" + strings.Join(caseSqls, " OR ") + ") "
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") VALUES (" + strings.Join(insertVals, ",") + ") "
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(upds, ",")

	// 执行批量insert sql
	res, err := od.dc.TxExec(tx, sqlTemp, args...)
	if err != nil {
		logx.Errorf("执行sql失败：%s, sql: [ %s ]", err.Error(), sqlTemp)
	}
	return res, err
}

func (od *OracleDialect) GetDataConverter() dbi.DataConverter {
	return converter
}

var (
	// 数字类型
	numberTypeRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`)
	// 日期时间类型
	datetimeTypeRegexp = regexp.MustCompile(`(?i)date|timestamp`)

	converter = new(DataConverter)
)

type DataConverter struct {
}

func (dc *DataConverter) GetDataType(dbColumnType string) dbi.DataType {
	if numberTypeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	// 日期时间类型
	if datetimeTypeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	return dbi.DataTypeString
}

func (dc *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
	str := anyx.ToString(dbColumnValue)
	switch dataType {
	// oracle把日期类型数据格式化输出
	case dbi.DataTypeDateTime: // "2024-01-02T22:08:22.275697+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.DateTime)
	}
	return str
}

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	// oracle把日期类型的数据转化为time类型
	if dataType == dbi.DataTypeDateTime {
		res, _ := time.Parse(time.RFC3339, anyx.ConvString(dbColumnValue))
		return res
	}
	return dbColumnValue
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
