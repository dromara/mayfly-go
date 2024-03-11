package dm

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"strings"
	"time"

	"github.com/kanzihuang/vitess/go/vt/sqlparser"

	_ "gitee.com/chunanyong/dm"
)

type DMDialect struct {
	dc *dbi.DbConn
}

func (dd *DMDialect) GetMetaData() dbi.MetaData {
	return &DMMetaData{
		dc: dd.dc,
	}
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (dd *DMDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, fmt.Errorf("该数据库类型不支持数据库备份与恢复: %v", dd.dc.Info.Type)
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

func (dd *DMDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
	// 执行批量insert sql
	// insert into "table_name" ("column1", "column2", ...) values (value1, value2, ...)

	dbType := dd.dc.Info.Type

	// 无需处理重复数据，直接执行批量insert
	if duplicateStrategy == dbi.DuplicateStrategyNone || duplicateStrategy == 0 {
		return dd.batchInsertSimple(dbType, tx, tableName, columns, values)
	} else { // 执行MERGE INTO语句
		return dd.batchInsertMerge(dbType, tx, tableName, columns, values)
	}
}

func (dd *DMDialect) batchInsertSimple(dbType dbi.DbType, tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	sqlTemp := fmt.Sprintf("insert into %s (%s) values %s", dbType.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)
	effRows := 0
	for _, value := range values {
		// 达梦数据库只能一条条的执行insert
		res, err := dd.dc.TxExec(tx, sqlTemp, value...)
		if err != nil {
			logx.Errorf("执行sql失败：%s, sql: [ %s ]", err.Error(), sqlTemp)
		}
		effRows += int(res)
	}
	// 执行批量insert sql
	return int64(effRows), nil
}

func (dd *DMDialect) batchInsertMerge(dbType dbi.DbType, tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	// 查询主键字段
	uniqueCols := make([]string, 0)
	caseSqls := make([]string, 0)
	metadata := dd.GetMetaData()
	tableCols, _ := metadata.GetColumns(tableName)
	identityCols := make([]string, 0)
	for _, col := range tableCols {
		if col.IsPrimaryKey {
			uniqueCols = append(uniqueCols, col.ColumnName)
			caseSqls = append(caseSqls, fmt.Sprintf("( T1.%s = T2.%s )", dbType.QuoteIdentifier(col.ColumnName), dbType.QuoteIdentifier(col.ColumnName)))
		}
		if col.IsIdentity {
			// 自增字段不放入insert内，即使是设置了identity_insert on也不起作用
			identityCols = append(identityCols, dbType.QuoteIdentifier(col.ColumnName))
		}
	}
	// 查询唯一索引涉及到的字段，并组装到match条件内
	indexs, _ := metadata.GetTableIndex(tableName)
	if indexs != nil {
		for _, index := range indexs {
			if index.IsUnique {
				cols := strings.Split(index.ColumnName, ",")
				tmp := make([]string, 0)
				for _, col := range cols {
					uniqueCols = append(uniqueCols, col)
					tmp = append(tmp, fmt.Sprintf(" T1.%s = T2.%s ", dbType.QuoteIdentifier(col), dbType.QuoteIdentifier(col)))
				}
				caseSqls = append(caseSqls, fmt.Sprintf("( %s )", strings.Join(tmp, " AND ")))
			}
		}
	}

	// 重复数据处理策略
	phs := make([]string, 0)
	insertVals := make([]string, 0)
	upds := make([]string, 0)
	insertCols := make([]string, 0)
	for _, column := range columns {
		phs = append(phs, fmt.Sprintf("? %s", column))
		if !collx.ArrayContains(uniqueCols, dbType.RemoveQuote(column)) {
			upds = append(upds, fmt.Sprintf("T1.%s = T2.%s", column, column))
		}
		if !collx.ArrayContains(identityCols, column) {
			insertCols = append(insertCols, fmt.Sprintf("%s", column))
			insertVals = append(insertVals, fmt.Sprintf("T2.%s", column))
		}

	}
	t2s := make([]string, 0)
	for i := 0; i < len(values); i++ {
		t2s = append(t2s, fmt.Sprintf("SELECT %s FROM dual", strings.Join(phs, ",")))
	}
	t2 := strings.Join(t2s, " UNION ALL ")

	sqlTemp := "MERGE INTO " + dbType.QuoteIdentifier(tableName) + " T1 USING (" + t2 + ") T2 ON " + strings.Join(caseSqls, " OR ")
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") VALUES (" + strings.Join(insertVals, ",") + ")"
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(upds, ",")

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}

	return dd.dc.TxExec(tx, sqlTemp, args...)
}

func (dd *DMDialect) GetDataConverter() dbi.DataConverter {
	return converter
}

type DataConverter struct {
}

func (dd *DataConverter) GetDataType(dbColumnType string) dbi.DataType {
	if numberRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	if datetimeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	if dateRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDate
	}
	if timeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeTime
	}
	return dbi.DataTypeString
}

func (dd *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
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

func (dd *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	// 如果dataType是datetime而dbColumnValue是string类型，则需要转换为time.Time类型
	_, ok := dbColumnValue.(string)
	if ok {
		if dataType == dbi.DataTypeDateTime {
			res, _ := time.Parse(time.RFC3339, anyx.ToString(dbColumnValue))
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

func (dd *DMDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName
	metadata := dd.GetMetaData()
	ddl, err := metadata.GetTableDDL(tableName)
	if err != nil {
		return err
	}
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// 替换新表名
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("\"%s\"", strings.ToUpper(tableName)), fmt.Sprintf("\"%s\"", strings.ToUpper(newTableName)))
	// 去除空格换行
	ddl = stringx.TrimSpaceAndBr(ddl)
	sqls, err := sqlparser.SplitStatementToPieces(ddl, sqlparser.WithDialect(dd.dc.Info.Type.Dialect()))
	for _, sql := range sqls {
		_, _ = dd.dc.Exec(sql)
	}

	// 复制数据
	if copy.CopyData {
		go func() {
			// 设置允许填充自增列之后，显示指定列名可以插入自增列
			_, _ = dd.dc.Exec(fmt.Sprintf("set identity_insert \"%s\" on", newTableName))
			// 获取列名
			columns, _ := metadata.GetColumns(tableName)
			columnArr := make([]string, 0)
			for _, column := range columns {
				columnArr = append(columnArr, fmt.Sprintf("\"%s\"", column.ColumnName))
			}
			columnStr := strings.Join(columnArr, ",")
			// 插入新数据并显示指定列
			_, _ = dd.dc.Exec(fmt.Sprintf("insert into \"%s\" (%s) select %s from \"%s\"", newTableName, columnStr, columnStr, tableName))

			// 执行完成后关闭允许填充自增列
			_, _ = dd.dc.Exec(fmt.Sprintf("set identity_insert \"%s\" off", newTableName))
		}()
	}

	return err
}
