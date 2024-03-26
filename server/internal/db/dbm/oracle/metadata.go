package oracle

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"strings"
	"time"

	"github.com/may-fly/cast"
)

// ---------------------------------- DM元数据 -----------------------------------
const (
	ORACLE_META_FILE      = "metasql/oracle_meta.sql"
	ORACLE_DB_SCHEMAS     = "ORACLE_DB_SCHEMAS"
	ORACLE_TABLE_INFO_KEY = "ORACLE_TABLE_INFO"
	ORACLE_INDEX_INFO_KEY = "ORACLE_INDEX_INFO"
	ORACLE_COLUMN_MA_KEY  = "ORACLE_COLUMN_MA"
)

type OracleMetaData struct {
	dbi.DefaultMetaData

	dc *dbi.DbConn
}

func (od *OracleMetaData) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := od.dc.Query("select * from v$instance")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["VERSION"]),
	}
	return ds, nil
}

func (od *OracleMetaData) GetDbNames() ([]string, error) {
	_, res, err := od.dc.Query("SELECT name AS DBNAME FROM v$database")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, cast.ToString(re["DBNAME"]))
	}

	return databases, nil
}

func (od *OracleMetaData) GetTables(tableNames ...string) ([]dbi.Table, error) {
	meta := od.dc.GetMetaData()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", meta.RemoveQuote(val))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = od.dc.Query(sql)
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    cast.ToString(re["TABLE_NAME"]),
			TableComment: cast.ToString(re["TABLE_COMMENT"]),
			CreateTime:   cast.ToString(re["CREATE_TIME"]),
			TableRows:    cast.ToInt(re["TABLE_ROWS"]),
			DataLength:   cast.ToInt64(re["DATA_LENGTH"]),
			IndexLength:  cast.ToInt64(re["INDEX_LENGTH"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (od *OracleMetaData) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	meta := od.dc.GetMetaData()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", meta.RemoveQuote(val))
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
		column := dbi.Column{
			TableName:     cast.ToString(re["TABLE_NAME"]),
			ColumnName:    cast.ToString(re["COLUMN_NAME"]),
			DataType:      dbi.ColumnDataType(cast.ToString(re["DATA_TYPE"])),
			CharMaxLength: cast.ToInt(re["CHAR_MAX_LENGTH"]),
			ColumnComment: cast.ToString(re["COLUMN_COMMENT"]),
			Nullable:      cast.ToString(re["NULLABLE"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["IS_PRIMARY_KEY"]) == 1,
			IsIdentity:    cast.ToInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: cast.ToString(re["COLUMN_DEFAULT"]),
			NumPrecision:  cast.ToInt(re["NUM_PRECISION"]),
			NumScale:      cast.ToInt(re["NUM_SCALE"]),
		}

		od.FixColumn(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

func (od *OracleMetaData) FixColumn(column *dbi.Column) {
	// 如果默认值包含.nextval，说明是序列，默认值为null
	if strings.Contains(column.ColumnDefault, ".nextval") {
		column.ColumnDefault = ""
	}

	// 统一处理一下数据类型的长度
	if collx.ArrayAnyMatches([]string{"date", "time", "lob", "int"}, strings.ToLower(string(column.DataType))) {
		// 如果是不需要设置长度的类型
		column.CharMaxLength = 0
		column.NumPrecision = 0
	} else if strings.Contains(strings.ToLower(string(column.DataType)), "char") {
		// 如果是字符串类型，长度最大4000，否则修改字段类型为clob
		if column.CharMaxLength > 4000 {
			column.DataType = "NCLOB"
			column.CharMaxLength = 0
			column.NumPrecision = 0
		}
	}
}

func (od *OracleMetaData) GetPrimaryKey(tablename string) (string, error) {
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
func (od *OracleMetaData) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := od.dc.Query(fmt.Sprintf(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    cast.ToString(re["INDEX_NAME"]),
			ColumnName:   cast.ToString(re["COLUMN_NAME"]),
			IndexType:    cast.ToString(re["INDEX_TYPE"]),
			IndexComment: cast.ToString(re["INDEX_COMMENT"]),
			IsUnique:     cast.ToInt(re["IS_UNIQUE"]) == 1,
			SeqInIndex:   cast.ToInt(re["SEQ_IN_INDEX"]),
			IsPrimaryKey: cast.ToInt(re["IS_PRIMARY"]) == 1,
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

// 获取建索引ddl
func (od *OracleMetaData) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {

	meta := od.dc.GetMetaData()
	sqls := make([]string, 0)
	comments := make([]string, 0)

	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}

		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, name := range cols {
			colNames[i] = meta.QuoteIdentifier(name)
		}

		sqls = append(sqls, fmt.Sprintf("CREATE %s INDEX %s ON %s(%s)", unique, index.IndexName, meta.QuoteIdentifier(tableInfo.TableName), strings.Join(colNames, ",")))
	}

	sqlArr := make([]string, 0)

	sqlArr = append(sqlArr, sqls...)

	if len(comments) > 0 {
		sqlArr = append(sqlArr, comments...)
	}

	return sqlArr
}

func (od *OracleMetaData) genColumnBasicSql(column dbi.Column) string {
	meta := od.dc.GetMetaData()
	colName := meta.QuoteIdentifier(column.ColumnName)

	if column.IsIdentity {
		// 如果是自增，不需要设置默认值和空值，自增列数据类型必须是number
		return fmt.Sprintf(" %s NUMBER generated by default as IDENTITY", colName)
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := ""
	if column.ColumnDefault != "" {
		defVal = fmt.Sprintf(" DEFAULT %v", column.ColumnDefault)
	}

	columnSql := fmt.Sprintf(" %s %s%s%s", colName, column.GetColumnType(), defVal, nullAble)
	return columnSql
}

// 获取建表ddl
func (od *OracleMetaData) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	meta := od.dc.GetMetaData()
	quoteTableName := meta.QuoteIdentifier(tableInfo.TableName)
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		dropSqlTmp := `
declare
      num number;
begin
    select count(1) into num from user_tables where table_name = '%s' and owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual) ;
    if num > 0 then
        execute immediate 'drop table "%s"' ;
    end if;
end`
		sqlArr = append(sqlArr, fmt.Sprintf(dropSqlTmp, tableInfo.TableName, tableInfo.TableName))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s ( \n", quoteTableName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)
	// 把通用类型转换为达梦类型
	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, meta.QuoteIdentifier(column.ColumnName))
		}
		fields = append(fields, od.genColumnBasicSql(column))
		// 防止注释内含有特殊字符串导致sql出错
		if column.ColumnComment != "" {
			comment := meta.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf("COMMENT ON COLUMN %s.%s IS '%s'", quoteTableName, meta.QuoteIdentifier(column.ColumnName), comment))
		}
	}

	// 建表
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"
	sqlArr = append(sqlArr, createSql)

	// 表注释
	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		tableCommentSql = fmt.Sprintf("COMMENT ON TABLE %s is '%s'", meta.QuoteIdentifier(tableInfo.TableName), meta.QuoteEscape(tableInfo.TableComment))
		sqlArr = append(sqlArr, tableCommentSql)
	}

	// 列注释
	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}

	return sqlArr
}

// 获取建表ddl
func (od *OracleMetaData) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {

	// 1.获取表信息
	tbs, err := od.GetTables(tableName)
	tableInfo := &dbi.Table{}
	if err != nil || tbs == nil || len(tbs) <= 0 {
		logx.Errorf("获取表信息失败, %s", tableName)
		return "", err
	}
	tableInfo.TableName = tbs[0].TableName
	tableInfo.TableComment = tbs[0].TableComment

	// 2.获取列信息
	columns, err := od.GetColumns(tableName)
	if err != nil {
		logx.Errorf("获取列信息失败, %s", tableName)
		return "", err
	}
	tableDDLArr := od.GenerateTableDDL(columns, *tableInfo, dropBeforeCreate)
	// 3.获取索引信息
	indexs, err := od.GetTableIndex(tableName)
	if err != nil {
		logx.Errorf("获取索引信息失败, %s", tableName)
		return "", err
	}
	// 组装返回
	tableDDLArr = append(tableDDLArr, od.GenerateIndexDDL(indexs, *tableInfo)...)
	return strings.Join(tableDDLArr, ";\n"), nil
}

// 获取DM当前连接的库可访问的schemaNames
func (od *OracleMetaData) GetSchemas() ([]string, error) {
	sql := dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_DB_SCHEMAS)
	_, res, err := od.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, cast.ToString(re["USERNAME"]))
	}
	return schemaNames, nil
}

func (od *OracleMetaData) GetDataConverter() dbi.DataConverter {
	return converter
}

var (
	// 数字类型
	numberTypeRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`)
	// 日期时间类型
	datetimeTypeRegexp = regexp.MustCompile(`(?i)date|timestamp`)

	bracketsRegexp = regexp.MustCompile(`\((\d+)\)`)

	converter = new(DataConverter)

	// oracle数据类型 映射 公共数据类型
	commonColumnTypeMap = map[string]dbi.ColumnDataType{
		"CHAR":          dbi.CommonTypeChar,
		"NCHAR":         dbi.CommonTypeChar,
		"VARCHAR2":      dbi.CommonTypeVarchar,
		"NVARCHAR2":     dbi.CommonTypeVarchar,
		"NUMBER":        dbi.CommonTypeNumber,
		"INTEGER":       dbi.CommonTypeInt,
		"INT":           dbi.CommonTypeInt,
		"DECIMAL":       dbi.CommonTypeNumber,
		"FLOAT":         dbi.CommonTypeNumber,
		"REAL":          dbi.CommonTypeNumber,
		"BINARY_FLOAT":  dbi.CommonTypeNumber,
		"BINARY_DOUBLE": dbi.CommonTypeNumber,
		"DATE":          dbi.CommonTypeDate,
		"TIMESTAMP":     dbi.CommonTypeDatetime,
		"LONG":          dbi.CommonTypeLongtext,
		"BLOB":          dbi.CommonTypeLongtext,
		"CLOB":          dbi.CommonTypeLongtext,
		"NCLOB":         dbi.CommonTypeLongtext,
		"BFILE":         dbi.CommonTypeBinary,
	}

	// 公共数据类型 映射 oracle数据类型
	oracleColumnTypeMap = map[dbi.ColumnDataType]string{
		dbi.CommonTypeVarchar:    "NVARCHAR2",
		dbi.CommonTypeChar:       "NCHAR",
		dbi.CommonTypeText:       "CLOB",
		dbi.CommonTypeBlob:       "CLOB",
		dbi.CommonTypeLongblob:   "CLOB",
		dbi.CommonTypeLongtext:   "CLOB",
		dbi.CommonTypeBinary:     "BFILE",
		dbi.CommonTypeMediumblob: "CLOB",
		dbi.CommonTypeMediumtext: "CLOB",
		dbi.CommonTypeVarbinary:  "BFILE",
		dbi.CommonTypeInt:        "INT",
		dbi.CommonTypeSmallint:   "INT",
		dbi.CommonTypeTinyint:    "INT",
		dbi.CommonTypeNumber:     "NUMBER",
		dbi.CommonTypeBigint:     "NUMBER",
		dbi.CommonTypeDatetime:   "DATE",
		dbi.CommonTypeDate:       "DATE",
		dbi.CommonTypeTime:       "DATE",
		dbi.CommonTypeTimestamp:  "TIMESTAMP",
		dbi.CommonTypeEnum:       "CLOB",
		dbi.CommonTypeJSON:       "CLOB",
	}
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
		// 尝试用时间格式解析
		res, err := time.Parse(time.DateTime, str)
		if err == nil {
			return str
		}
		res, _ = time.Parse(time.RFC3339, str)
		return res.Format(time.DateTime)
	}
	return str
}

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	// oracle把日期类型的数据转化为time类型
	if dataType == dbi.DataTypeDateTime {
		res, _ := time.Parse(time.RFC3339, cast.ToString(dbColumnValue))
		return res
	}
	return dbColumnValue
}

func (dc *DataConverter) WrapValue(dbColumnValue any, dataType dbi.DataType) string {
	if dbColumnValue == nil {
		return "NULL"
	}
	switch dataType {
	case dbi.DataTypeNumber:
		return fmt.Sprintf("%v", dbColumnValue)
	case dbi.DataTypeString:
		val := fmt.Sprintf("%v", dbColumnValue)
		// 转义单引号
		val = strings.Replace(val, `'`, `''`, -1)
		val = strings.Replace(val, `\''`, `\'`, -1)
		// 转义换行符
		val = strings.Replace(val, "\n", "\\n", -1)
		return fmt.Sprintf("'%s'", val)
	case dbi.DataTypeDate, dbi.DataTypeDateTime, dbi.DataTypeTime:
		return fmt.Sprintf("to_timestamp('%s', 'yyyy-mm-dd hh24:mi:ss')", dc.FormatData(dbColumnValue, dataType))
	}
	return fmt.Sprintf("'%s'", dbColumnValue)
}
