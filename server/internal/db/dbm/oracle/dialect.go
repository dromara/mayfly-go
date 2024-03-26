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

func (od *OracleDialect) ToCommonColumn(dialectColumn *dbi.Column) {
	// 翻译为通用数据库类型
	dataType := dialectColumn.DataType
	t1 := commonColumnTypeMap[string(dataType)]
	if t1 == "" {
		dialectColumn.DataType = dbi.CommonTypeVarchar
		dialectColumn.CharMaxLength = 2000
	} else {
		dialectColumn.DataType = t1
		// 如果是number类型，需要根据公共类型加上长度, 如 bigint 需要转换为number(19,0)
		if strings.Contains(string(t1), "NUMBER") {
			dialectColumn.CharMaxLength = 19
		}
	}
}

func (od *OracleDialect) ToColumn(commonColumn *dbi.Column) {
	ctype := oracleColumnTypeMap[commonColumn.DataType]
	if ctype == "" {
		commonColumn.DataType = "NVARCHAR2"
		commonColumn.CharMaxLength = 2000
	} else {
		commonColumn.DataType = dbi.ColumnDataType(ctype)
		od.dc.GetMetaData().FixColumn(commonColumn)
	}
}

func (od *OracleDialect) CreateTable(commonColumns []dbi.Column, tableInfo dbi.Table, dropOldTable bool) (int, error) {
	meta := od.dc.GetMetaData()
	sqlArr := meta.GenerateTableDDL(commonColumns, tableInfo, dropOldTable)
	// 需要分开执行sql
	for _, sqlStr := range sqlArr {
		_, err := od.dc.Exec(sqlStr)
		if err != nil {
			return 0, err
		}
	}
	return len(sqlArr), nil
}

func (od *OracleDialect) CreateIndex(tableInfo dbi.Table, indexs []dbi.Index) error {
	meta := od.dc.GetMetaData()
	sqlArr := meta.GenerateIndexDDL(indexs, tableInfo)
	_, err := od.dc.Exec(strings.Join(sqlArr, ";"))
	return err
}
