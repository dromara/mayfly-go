package postgres

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

const (
	PGSQL_META_FILE      = "metasql/pgsql_meta.sql"
	PGSQL_DB_SCHEMAS     = "PGSQL_DB_SCHEMAS"
	PGSQL_TABLE_INFO_KEY = "PGSQL_TABLE_INFO"
	PGSQL_INDEX_INFO_KEY = "PGSQL_INDEX_INFO"
	PGSQL_COLUMN_MA_KEY  = "PGSQL_COLUMN_MA"
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
		Version: cast.ToString(res[0]["server_version"]),
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
		databases = append(databases, cast.ToString(re["dbname"]))
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

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = pd.dc.Query(sql)
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    re["tableName"].(string),
			TableComment: cast.ToString(re["tableComment"]),
			CreateTime:   cast.ToString(re["createTime"]),
			TableRows:    cast.ToInt(re["tableRows"]),
			DataLength:   cast.ToInt64(re["dataLength"]),
			IndexLength:  cast.ToInt64(re["indexLength"]),
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
		column := dbi.Column{
			TableName:     cast.ToString(re["tableName"]),
			ColumnName:    cast.ToString(re["columnName"]),
			DataType:      dbi.ColumnDataType(cast.ToString(re["dataType"])),
			CharMaxLength: cast.ToInt(re["charMaxLength"]),
			ColumnComment: cast.ToString(re["columnComment"]),
			Nullable:      cast.ToString(re["nullable"]),
			IsPrimaryKey:  cast.ToInt(re["isPrimaryKey"]) == 1,
			IsIdentity:    cast.ToInt(re["isIdentity"]) == 1,
			ColumnDefault: cast.ToString(re["columnDefault"]),
			NumPrecision:  cast.ToInt(re["numPrecision"]),
			NumScale:      cast.ToInt(re["numScale"]),
		}

		columns = append(columns, column)
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
			IndexName:    cast.ToString(re["indexName"]),
			ColumnName:   cast.ToString(re["columnName"]),
			IndexType:    cast.ToString(re["IndexType"]),
			IndexComment: cast.ToString(re["indexComment"]),
			IsUnique:     cast.ToInt(re["isUnique"]) == 1,
			SeqInIndex:   cast.ToInt(re["seqInIndex"]),
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

func (pd *PgsqlMetaData) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
	creates := make([]string, 0)
	drops := make([]string, 0)
	comments := make([]string, 0)
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

		// 如果索引名存在，先删除索引
		drops = append(drops, fmt.Sprintf("drop index if exists %s.%s", pd.dc.Info.CurrentSchema(), indexName))

		// 创建索引
		creates = append(creates, fmt.Sprintf("CREATE %s INDEX %s on %s.%s(%s)", unique, indexName, pd.dc.Info.CurrentSchema(), tableInfo.TableName, index.ColumnName))
		if index.IndexComment != "" {
			comments = append(comments, fmt.Sprintf("COMMENT ON INDEX %s.%s IS '%s'", pd.dc.Info.CurrentSchema(), indexName, index.IndexComment))
		}
	}

	sqlArr := make([]string, 0)

	if len(drops) > 0 {
		sqlArr = append(sqlArr, drops...)
	}

	if len(creates) > 0 {
		sqlArr = append(sqlArr, creates...)
	}
	if len(comments) > 0 {
		sqlArr = append(sqlArr, comments...)
	}
	return sqlArr
}

func (pd *PgsqlMetaData) genColumnBasicSql(column dbi.Column) string {
	meta := pd.dc.GetMetaData()
	colName := meta.QuoteIdentifier(column.ColumnName)
	dataType := string(column.DataType)

	// 如果是自增类型，需要转换为serial
	if column.IsIdentity {
		if dataType == "int4" {
			column.DataType = "serial"
		} else if dataType == "int2" {
			column.DataType = "smallserial"
		} else if dataType == "int8" {
			column.DataType = "bigserial"
		} else {
			column.DataType = "bigserial"
		}

		return fmt.Sprintf(" %s %s NOT NULL", colName, column.GetColumnType())
	}

	nullAble := ""
	if column.Nullable == "NO" {
		nullAble = " NOT NULL"
		// 如果字段不能为空，则设置默认值
		if column.ColumnDefault == "" {
			if collx.ArrayAnyMatches([]string{"char", "text", "lob"}, strings.ToLower(dataType)) {
				// 文本默认值为空字符串
				column.ColumnDefault = " "
			} else if collx.ArrayAnyMatches([]string{"int", "num"}, strings.ToLower(dataType)) {
				// 数字默认值为0
				column.ColumnDefault = "0"
			}
		}
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, dataType) {
			// 如果是文本类型，则默认值不能带括号
			if collx.ArrayAnyMatches([]string{"char", "text", "lob"}, dataType) {
				column.ColumnDefault = ""
			}

			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) &&
				collx.ArrayAnyMatches([]string{"DATE", "TIME"}, strings.ToUpper(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
		}
		// 如果数据类型是日期时间，则写死默认值函数
		if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) {
			column.ColumnDefault = "CURRENT_TIMESTAMP"
		}

		if column.ColumnDefault != "" {
			if mark {
				defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
			} else {
				defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
			}
		}
	}

	columnSql := fmt.Sprintf(" %s %s %s %s ", colName, column.GetColumnType(), nullAble, defVal)
	return columnSql
}

func (pd *PgsqlMetaData) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {

	meta := pd.dc.GetMetaData()
	replacer := strings.NewReplacer(";", "", "'", "")

	sqlArr := make([]string, 0)
	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", meta.QuoteIdentifier(tableInfo.TableName)))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", meta.QuoteIdentifier(tableInfo.TableName))
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)
	commentTmp := "comment on column %s.%s is '%s'"

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, meta.QuoteIdentifier(column.ColumnName))
		}

		fields = append(fields, pd.genColumnBasicSql(column))

		// 防止注释内含有特殊字符串导致sql出错
		if column.ColumnComment != "" {
			comment := replacer.Replace(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf(commentTmp, column.TableName, column.ColumnName, comment))
		}
	}

	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += ")"

	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		commentTmp := "comment on table %s is '%s'"
		tableCommentSql = fmt.Sprintf(commentTmp, tableInfo.TableName, replacer.Replace(tableInfo.TableComment))
	}

	// create
	sqlArr = append(sqlArr, createSql)

	// table comment
	if tableCommentSql != "" {
		sqlArr = append(sqlArr, tableCommentSql)
	}
	// column comment
	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}

	return sqlArr
}

// 获取建表ddl
func (pd *PgsqlMetaData) GetTableDDL(tableName string) (string, error) {

	// 1.获取表信息
	tbs, err := pd.GetTables(tableName)
	tableInfo := &dbi.Table{}
	if err != nil && len(tbs) > 0 {

		logx.Errorf("获取表信息失败, %s", tableName)
		return "", err
	}
	tableInfo.TableName = tbs[0].TableName
	tableInfo.TableComment = tbs[0].TableComment

	// 2.获取列信息
	columns, err := pd.GetColumns(tableName)
	if err != nil {
		logx.Errorf("获取列信息失败, %s", tableName)
		return "", err
	}
	tableDDLArr := pd.GenerateTableDDL(columns, *tableInfo, false)
	// 3.获取索引信息
	indexs, err := pd.GetTableIndex(tableName)
	if err != nil {
		logx.Errorf("获取索引信息失败, %s", tableName)
		return "", err
	}
	// 组装返回
	tableDDLArr = append(tableDDLArr, pd.GenerateIndexDDL(indexs, *tableInfo)...)
	return strings.Join(tableDDLArr, ";\n"), nil
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
		schemaNames = append(schemaNames, cast.ToString(re["schemaName"]))
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
	commonColumnTypeMap = map[string]dbi.ColumnDataType{
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
		"bytea":       dbi.CommonTypeText,
		"date":        dbi.CommonTypeDate,
		"time":        dbi.CommonTypeTime,
		"timestamp":   dbi.CommonTypeTimestamp,
	}
	// 公共数据类型 映射 pgsql数据类型
	pgsqlColumnTypeMap = map[dbi.ColumnDataType]string{
		dbi.CommonTypeVarchar:    "varchar",
		dbi.CommonTypeChar:       "char",
		dbi.CommonTypeText:       "text",
		dbi.CommonTypeBlob:       "text",
		dbi.CommonTypeLongblob:   "text",
		dbi.CommonTypeLongtext:   "text",
		dbi.CommonTypeBinary:     "text",
		dbi.CommonTypeMediumblob: "text",
		dbi.CommonTypeMediumtext: "text",
		dbi.CommonTypeVarbinary:  "text",
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
	return cast.ToString(dbColumnValue)
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
