package oracle

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"
	"time"

	_ "gitee.com/chunanyong/dm"
)

// ---------------------------------- DM元数据 -----------------------------------
const (
	ORACLE_META_FILE      = "metasql/oracle_meta.sql"
	ORACLE_DB_SCHEMAS     = "ORACLE_DB_SCHEMAS"
	ORACLE_TABLE_INFO_KEY = "ORACLE_TABLE_INFO"
	ORACLE_INDEX_INFO_KEY = "ORACLE_INDEX_INFO"
	ORACLE_COLUMN_MA_KEY  = "ORACLE_COLUMN_MA"
)

type OracleDialect struct {
	dc *dbi.DbConn
}

func (od *OracleDialect) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := od.dc.Query("select * from v$instance")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["VERSION"]),
	}
	return ds, nil
}

func (od *OracleDialect) GetDbNames() ([]string, error) {
	_, res, err := od.dc.Query("SELECT name AS DBNAME FROM v$database")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, anyx.ConvString(re["DBNAME"]))
	}

	return databases, nil
}

// 获取表基础元信息, 如表名等
func (od *OracleDialect) GetTables() ([]dbi.Table, error) {

	// 首先执行更新统计信息sql 这个统计信息在数据量比较大的时候就比较耗时，所以最好定时执行
	// _, _, err := pd.dc.Query("dbms_stats.GATHER_SCHEMA_stats(SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))")

	// 查询表信息
	_, res, err := od.dc.Query(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_TABLE_INFO_KEY))
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    anyx.ConvString(re["TABLE_NAME"]),
			TableComment: anyx.ConvString(re["TABLE_COMMENT"]),
			CreateTime:   anyx.ConvString(re["CREATE_TIME"]),
			TableRows:    anyx.ConvInt(re["TABLE_ROWS"]),
			DataLength:   anyx.ConvInt64(re["DATA_LENGTH"]),
			IndexLength:  anyx.ConvInt64(re["INDEX_LENGTH"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (od *OracleDialect) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dbType := od.dc.Info.Type
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbType.RemoveQuote(val))
	}), ",")

	// 如果表数量超过了1000，需要分批查询
	if len(tableNames) > 1000 {
		columns := make([]dbi.Column, 0)
		for i := 0; i < len(tableNames); i += 1000 {
			end := i + 1000
			if end > len(tableNames) {
				end = len(tableNames)
			}
			tables := tableNames[i:end]
			cols, err := od.GetColumns(tables...)
			if err != nil {
				return nil, err
			}
			columns = append(columns, cols...)
		}
		return columns, nil
	}

	_, res, err := od.dc.Query(fmt.Sprintf(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		defaultVal := anyx.ConvString(re["COLUMN_DEFAULT"])
		// 如果默认值包含.nextval，说明是序列，默认值为null
		if strings.Contains(defaultVal, ".nextval") {
			defaultVal = ""
		}
		columns = append(columns, dbi.Column{
			TableName:     anyx.ConvString(re["TABLE_NAME"]),
			ColumnName:    anyx.ConvString(re["COLUMN_NAME"]),
			ColumnType:    anyx.ConvString(re["COLUMN_TYPE"]),
			ColumnComment: anyx.ConvString(re["COLUMN_COMMENT"]),
			Nullable:      anyx.ConvString(re["NULLABLE"]),
			IsPrimaryKey:  anyx.ConvInt(re["IS_PRIMARY_KEY"]) == 1,
			IsIdentity:    anyx.ConvInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: defaultVal,
			NumScale:      anyx.ConvString(re["NUM_SCALE"]),
		})
	}
	return columns, nil
}

func (od *OracleDialect) GetPrimaryKey(tablename string) (string, error) {
	columns, err := od.GetColumns(tablename)
	if err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", errorx.NewBiz("[%s] 表不存在", tablename)
	}
	for _, v := range columns {
		if v.IsPrimaryKey {
			return v.ColumnName, nil
		}
	}

	return columns[0].ColumnName, nil
}

// 获取表索引信息
func (od *OracleDialect) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := od.dc.Query(fmt.Sprintf(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    anyx.ConvString(re["INDEX_NAME"]),
			ColumnName:   anyx.ConvString(re["COLUMN_NAME"]),
			IndexType:    anyx.ConvString(re["INDEX_TYPE"]),
			IndexComment: anyx.ConvString(re["INDEX_COMMENT"]),
			IsUnique:     anyx.ConvInt(re["IS_UNIQUE"]) == 1,
			SeqInIndex:   anyx.ConvInt(re["SEQ_IN_INDEX"]),
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	result := make([]dbi.Index, 0)
	key := ""
	for _, v := range indexs {
		// 当前的索引名
		in := v.IndexName
		if key == in {
			// 索引字段已根据名称和顺序排序，故取最后一个即可
			i := len(result) - 1
			// 同索引字段以逗号连接
			result[i].ColumnName = result[i].ColumnName + "," + v.ColumnName
		} else {
			key = in
			result = append(result, v)
		}
	}
	return result, nil
}

// 获取建表ddl
func (od *OracleDialect) GetTableDDL(tableName string) (string, error) {
	ddlSql := fmt.Sprintf("SELECT DBMS_METADATA.GET_DDL('TABLE', '%s', (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual)) AS TABLE_DDL FROM DUAL", tableName)
	_, res, err := od.dc.Query(ddlSql)
	if err != nil {
		return "", err
	}
	// 建表ddl
	var builder strings.Builder
	for _, re := range res {
		builder.WriteString(anyx.ConvString(re["TABLE_DDL"]))
	}

	// 表注释
	_, res, err = od.dc.Query(fmt.Sprintf(`
			select OWNER, COMMENTS from ALL_TAB_COMMENTS where TABLE_TYPE='TABLE' and TABLE_NAME = '%s'
		    and owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual) `, tableName))
	if err != nil {
		return "", err
	}
	for _, re := range res {
		// COMMENT ON TABLE "SYS_MENU" IS '菜单表';
		if re["COMMENTS"] != nil {
			tableComment := fmt.Sprintf("\n\nCOMMENT ON TABLE \"%s\".\"%s\" IS '%s';", re["OWNER"].(string), tableName, re["COMMENTS"].(string))
			builder.WriteString(tableComment)
		}
	}

	// 字段注释
	fieldSql := fmt.Sprintf(`
		SELECT OWNER, COLUMN_NAME, COMMENTS
		FROM ALL_COL_COMMENTS
		WHERE OWNER = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual)
		  AND TABLE_NAME = '%s'
		`, tableName)
	_, res, err = od.dc.Query(fieldSql)
	if err != nil {
		return "", err
	}

	builder.WriteString("\n")
	for _, re := range res {
		// COMMENT ON COLUMN "SYS_MENU"."BIZ_CODE" IS '业务编码，应用编码1';
		if re["COMMENTS"] != nil {
			fieldComment := fmt.Sprintf("\nCOMMENT ON COLUMN \"%s\".\"%s\".\"%s\" IS '%s';", re["OWNER"].(string), tableName, re["COLUMN_NAME"].(string), re["COMMENTS"].(string))
			builder.WriteString(fieldComment)
		}
	}

	// 索引信息
	indexSql := fmt.Sprintf(`
		select DBMS_METADATA.GET_DDL('INDEX', a.INDEX_NAME, a.OWNER) AS INDEX_DEF from ALL_INDEXES a
		join ALL_objects b on a.owner = b.owner and b.object_name = a.index_name and b.object_type = 'INDEX'
		where a.owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual)
		and a.table_name = '%s' 
	`, tableName)
	_, res, err = od.dc.Query(indexSql)
	if err != nil {
		return "", err
	}
	for _, re := range res {
		builder.WriteString("\n\n" + anyx.ConvString(re["INDEX_DEF"]))
	}

	return builder.String(), nil
}

// 获取DM当前连接的库可访问的schemaNames
func (od *OracleDialect) GetSchemas() ([]string, error) {
	sql := dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_DB_SCHEMAS)
	_, res, err := od.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, anyx.ConvString(re["USERNAME"]))
	}
	return schemaNames, nil
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
		indexs, _ := od.GetTableIndex(tableName)
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
	indexs, _ := od.GetTableIndex(tableName)
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
