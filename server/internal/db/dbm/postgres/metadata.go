package postgres

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"
	"time"
)

const (
	PGSQL_META_FILE               = "metasql/pgsql_meta.sql"
	PGSQL_DB_SCHEMAS              = "PGSQL_DB_SCHEMAS"
	PGSQL_TABLE_INFO_KEY          = "PGSQL_TABLE_INFO"
	PGSQL_INDEX_INFO_KEY          = "PGSQL_INDEX_INFO"
	PGSQL_COLUMN_MA_KEY           = "PGSQL_COLUMN_MA"
	PGSQL_TABLE_DDL_KEY           = "PGSQL_TABLE_DDL_FUNC"
	PGSQL_TABLE_INFO_BY_NAMES_KEY = "PGSQL_TABLE_INFO_BY_NAMES"
)

type PgsqlMetaData struct {
	dbi.DefaultMetaData

	dc *dbi.DbConn
}

func (pd *PgsqlMetaData) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := pd.dc.Query("SELECT version() as server_version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["server_version"]),
	}
	return ds, nil
}

func (pd *PgsqlMetaData) GetDbNames() ([]string, error) {
	_, res, err := pd.dc.Query("SELECT datname AS dbname FROM pg_database WHERE datistemplate = false AND has_database_privilege(datname, 'CONNECT')")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, anyx.ConvString(re["dbname"]))
	}

	return databases, nil
}

func (pd *PgsqlMetaData) GetTables(tableNames ...string) ([]dbi.Table, error) {
	meta := pd.dc.GetMetaData()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", meta.RemoveQuote(val))
	}), ",")

	var res []map[string]any
	var err error

	if tableNames != nil || len(tableNames) > 0 {
		_, res, err = pd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_INFO_BY_NAMES_KEY), names))
	} else {
		_, res, err = pd.dc.Query(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_INFO_KEY))
	}
	if err != nil {
		return nil, err
	}
	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    re["tableName"].(string),
			TableComment: anyx.ConvString(re["tableComment"]),
			CreateTime:   anyx.ConvString(re["createTime"]),
			TableRows:    anyx.ConvInt(re["tableRows"]),
			DataLength:   anyx.ConvInt64(re["dataLength"]),
			IndexLength:  anyx.ConvInt64(re["indexLength"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (pd *PgsqlMetaData) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	meta := pd.dc.GetMetaData()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", meta.RemoveQuote(val))
	}), ",")

	_, res, err := pd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		columns = append(columns, dbi.Column{
			TableName:     anyx.ConvString(re["tableName"]),
			ColumnName:    anyx.ConvString(re["columnName"]),
			ColumnType:    anyx.ConvString(re["columnType"]),
			ColumnComment: anyx.ConvString(re["columnComment"]),
			Nullable:      anyx.ConvString(re["nullable"]),
			IsPrimaryKey:  anyx.ConvInt(re["isPrimaryKey"]) == 1,
			IsIdentity:    anyx.ConvInt(re["isIdentity"]) == 1,
			ColumnDefault: anyx.ConvString(re["columnDefault"]),
			NumScale:      anyx.ConvString(re["numScale"]),
		})
	}
	return columns, nil
}

func (pd *PgsqlMetaData) GetPrimaryKey(tablename string) (string, error) {
	columns, err := pd.GetColumns(tablename)
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
func (pd *PgsqlMetaData) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := pd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    anyx.ConvString(re["indexName"]),
			ColumnName:   anyx.ConvString(re["columnName"]),
			IndexType:    anyx.ConvString(re["IndexType"]),
			IndexComment: anyx.ConvString(re["indexComment"]),
			IsUnique:     anyx.ConvInt(re["isUnique"]) == 1,
			SeqInIndex:   anyx.ConvInt(re["seqInIndex"]),
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
func (pd *PgsqlMetaData) GetTableDDL(tableName string) (string, error) {
	_, err := pd.dc.Exec(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_DDL_KEY))
	if err != nil {
		return "", err
	}

	ddlSql := fmt.Sprintf("select showcreatetable('%s','%s') as sql", pd.dc.Info.CurrentSchema(), tableName)
	_, res, err := pd.dc.Query(ddlSql)
	if err != nil {
		return "", err
	}

	return res[0]["sql"].(string), nil
}

// 获取pgsql当前连接的库可访问的schemaNames
func (pd *PgsqlMetaData) GetSchemas() ([]string, error) {
	sql := dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_DB_SCHEMAS)
	_, res, err := pd.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, anyx.ConvString(re["schemaName"]))
	}
	return schemaNames, nil
}

func (pd *PgsqlMetaData) DefaultDb() string {
	switch pd.dc.Info.Type {
	case dbi.DbTypePostgres, dbi.DbTypeGauss:
		return "postgres"
	case dbi.DbTypeKingbaseEs:
		return "security"
	case dbi.DbTypeVastbase:
		return "vastbase"
	default:
		return ""
	}
}

func (pd *PgsqlMetaData) GetDataConverter() dbi.DataConverter {
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
	// 定义正则表达式，匹配括号内的数字
	bracketsRegexp = regexp.MustCompile(`\((\d+)\)`)

	converter = new(DataConverter)

	// pgsql数据类型 映射 公共数据类型
	commonColumnTypeMap = map[string]string{
		"int2":        dbi.CommonTypeSmallint,
		"int4":        dbi.CommonTypeInt,
		"int8":        dbi.CommonTypeBigint,
		"numeric":     dbi.CommonTypeNumber,
		"decimal":     dbi.CommonTypeNumber,
		"smallserial": dbi.CommonTypeSmallint,
		"serial":      dbi.CommonTypeInt,
		"bigserial":   dbi.CommonTypeBigint,
		"largeserial": dbi.CommonTypeBigint,
		"money":       dbi.CommonTypeNumber,
		"bool":        dbi.CommonTypeTinyint,
		"char":        dbi.CommonTypeChar,
		"character":   dbi.CommonTypeChar,
		"nchar":       dbi.CommonTypeChar,
		"varchar":     dbi.CommonTypeVarchar,
		"text":        dbi.CommonTypeText,
		"bytea":       dbi.CommonTypeBinary,
		"date":        dbi.CommonTypeDate,
		"time":        dbi.CommonTypeTime,
		"timestamp":   dbi.CommonTypeTimestamp,
	}
	// 公共数据类型 映射 pgsql数据类型
	pgsqlColumnTypeMap = map[string]string{
		dbi.CommonTypeVarchar:    "varchar",
		dbi.CommonTypeChar:       "char",
		dbi.CommonTypeText:       "text",
		dbi.CommonTypeBlob:       "text",
		dbi.CommonTypeLongblob:   "text",
		dbi.CommonTypeLongtext:   "text",
		dbi.CommonTypeBinary:     "bytea",
		dbi.CommonTypeMediumblob: "text",
		dbi.CommonTypeMediumtext: "text",
		dbi.CommonTypeVarbinary:  "bytea",
		dbi.CommonTypeInt:        "int4",
		dbi.CommonTypeSmallint:   "int2",
		dbi.CommonTypeTinyint:    "int2",
		dbi.CommonTypeNumber:     "numeric",
		dbi.CommonTypeBigint:     "int8",
		dbi.CommonTypeDatetime:   "timestamp",
		dbi.CommonTypeDate:       "date",
		dbi.CommonTypeTime:       "time",
		dbi.CommonTypeTimestamp:  "timestamp",
		dbi.CommonTypeEnum:       "varchar(2000)",
		dbi.CommonTypeJSON:       "varchar(2000)",
	}
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
	str := fmt.Sprintf("%v", dbColumnValue)
	switch dataType {
	case dbi.DataTypeDateTime: // "2024-01-02T22:16:28.545377+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.DateTime)
	case dbi.DataTypeDate: //  "2024-01-02T00:00:00Z"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.DateOnly)
	case dbi.DataTypeTime: // "0000-01-01T22:16:28.545075+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.TimeOnly)
	}
	return anyx.ConvString(dbColumnValue)
}

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	// 如果dataType是datetime而dbColumnValue是string类型，则需要转换为time.Time类型
	_, ok := dbColumnValue.(string)
	if dataType == dbi.DataTypeDateTime && ok {
		res, _ := time.Parse(time.RFC3339, anyx.ToString(dbColumnValue))
		return res
	}
	if dataType == dbi.DataTypeDate && ok {
		res, _ := time.Parse(time.DateOnly, anyx.ToString(dbColumnValue))
		return res
	}
	if dataType == dbi.DataTypeTime && ok {
		res, _ := time.Parse(time.TimeOnly, anyx.ToString(dbColumnValue))
		return res
	}
	return dbColumnValue
}
