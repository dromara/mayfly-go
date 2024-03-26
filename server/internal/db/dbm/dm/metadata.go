package dm

import (
	"fmt"
	"io"
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
	DM_META_FILE      = "metasql/dm_meta.sql"
	DM_DB_SCHEMAS     = "DM_DB_SCHEMAS"
	DM_TABLE_INFO_KEY = "DM_TABLE_INFO"
	DM_INDEX_INFO_KEY = "DM_INDEX_INFO"
	DM_COLUMN_MA_KEY  = "DM_COLUMN_MA"
)

type DMMetaData struct {
	dbi.DefaultMetaData

	dc *dbi.DbConn
}

func (dd *DMMetaData) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := dd.dc.Query("select * from v$instance")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["SVR_VERSION"]),
	}
	return ds, nil
}

func (dd *DMMetaData) GetDbNames() ([]string, error) {
	_, res, err := dd.dc.Query("SELECT name AS DBNAME FROM v$database")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, cast.ToString(re["DBNAME"]))
	}

	return databases, nil
}

func (dd *DMMetaData) GetTables(tableNames ...string) ([]dbi.Table, error) {
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.RemoveQuote(dd, val))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(DM_META_FILE, DM_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = dd.dc.Query(sql)
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
func (dd *DMMetaData) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.RemoveQuote(dd, val))
	}), ",")

	_, res, err := dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		column := dbi.Column{
			TableName:     cast.ToString(re["TABLE_NAME"]),
			ColumnName:    cast.ToString(re["COLUMN_NAME"]),
			DataType:      dbi.ColumnDataType(anyx.ToString(re["DATA_TYPE"])),
			CharMaxLength: cast.ToInt(re["CHAR_MAX_LENGTH"]),
			ColumnComment: cast.ToString(re["COLUMN_COMMENT"]),
			Nullable:      cast.ToString(re["NULLABLE"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["IS_PRIMARY_KEY"]) == 1,
			IsIdentity:    cast.ToInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: cast.ToString(re["COLUMN_DEFAULT"]),
			NumPrecision:  cast.ToInt(re["NUM_PRECISION"]),
			NumScale:      cast.ToInt(re["NUM_SCALE"]),
		}
		dd.FixColumn(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

func (dd *DMMetaData) FixColumn(column *dbi.Column) {
	// 如果是date，不设长度
	if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(string(column.DataType))) {
		column.CharMaxLength = 0
		column.NumPrecision = 0
	} else
	// 如果是char且长度未设置，则默认长度2000
	if collx.ArrayAnyMatches([]string{"char"}, strings.ToLower(string(column.DataType))) && column.CharMaxLength == 0 {
		column.CharMaxLength = 2000
	}
}

func (dd *DMMetaData) GetPrimaryKey(tablename string) (string, error) {
	columns, err := dd.GetColumns(tablename)
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
func (dd *DMMetaData) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_INDEX_INFO_KEY), tableName))
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
			IsPrimaryKey: false,
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

func (dd *DMMetaData) genColumnBasicSql(column dbi.Column) string {
	meta := dd.dc.GetMetaData()
	colName := meta.QuoteIdentifier(column.ColumnName)
	dataType := string(column.DataType)

	incr := ""
	if column.IsIdentity {
		incr = " IDENTITY"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(dataType)) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) &&
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

	columnSql := fmt.Sprintf(" %s %s%s%s%s", colName, column.GetColumnType(), incr, nullAble, defVal)
	return columnSql

}

func (dd *DMMetaData) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
	meta := dd.dc.GetMetaData()
	sqls := make([]string, 0)
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

		sqls = append(sqls, fmt.Sprintf("create %s index %s on %s(%s)", unique, index.IndexName, meta.QuoteIdentifier(tableInfo.TableName), strings.Join(colNames, ",")))
	}
	return sqls
}

func (dd *DMMetaData) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	meta := dd.dc.GetMetaData()
	tbName := meta.QuoteIdentifier(tableInfo.TableName)
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("drop table if exists %s", tbName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("create table %s (", tbName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, meta.QuoteIdentifier(column.ColumnName))
		}
		fields = append(fields, dd.genColumnBasicSql(column))
		if column.ColumnComment != "" {
			comment := meta.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf("comment on column %s.%s is '%s'", tbName, meta.QuoteIdentifier(column.ColumnName), comment))
		}
	}
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(",\n PRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		comment := meta.QuoteEscape(tableInfo.TableComment)
		tableCommentSql = fmt.Sprintf("comment on table %s is '%s'", tbName, comment)
	}

	sqlArr = append(sqlArr, createSql)
	if tableCommentSql != "" {
		sqlArr = append(sqlArr, tableCommentSql)
	}

	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}

	return sqlArr
}

// 获取建表ddl
func (dd *DMMetaData) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {

	// 1.获取表信息
	tbs, err := dd.GetTables(tableName)
	tableInfo := &dbi.Table{}
	if err != nil || tbs == nil || len(tbs) <= 0 {
		logx.Errorf("获取表信息失败, %s", tableName)
		return "", err
	}
	tableInfo.TableName = tbs[0].TableName
	tableInfo.TableComment = tbs[0].TableComment

	// 2.获取列信息
	columns, err := dd.GetColumns(tableName)
	if err != nil {
		logx.Errorf("获取列信息失败, %s", tableName)
		return "", err
	}
	tableDDLArr := dd.GenerateTableDDL(columns, *tableInfo, dropBeforeCreate)
	// 3.获取索引信息
	indexs, err := dd.GetTableIndex(tableName)
	if err != nil {
		logx.Errorf("获取索引信息失败, %s", tableName)
		return "", err
	}
	// 组装返回
	tableDDLArr = append(tableDDLArr, dd.GenerateIndexDDL(indexs, *tableInfo)...)
	return strings.Join(tableDDLArr, ";\n"), nil
}

// 获取DM当前连接的库可访问的schemaNames
func (dd *DMMetaData) GetSchemas() ([]string, error) {
	sql := dbi.GetLocalSql(DM_META_FILE, DM_DB_SCHEMAS)
	_, res, err := dd.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, cast.ToString(re["SCHEMA_NAME"]))
	}
	return schemaNames, nil
}

func (dd *DMMetaData) BeforeDumpInsert(writer io.Writer, tableName string) {

}

func (dd *DMMetaData) BeforeDumpInsertSql(quoteSchema string, tableName string) string {
	return fmt.Sprintf("set identity_insert %s on;", tableName)
}

func (dd *DMMetaData) AfterDumpInsert(writer io.Writer, tableName string, columns []dbi.Column) {
	writer.Write([]byte("COMMIT;\n"))
}

func (dd *DMMetaData) GetDataConverter() dbi.DataConverter {
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

	// 达梦数据类型 对应 公共数据类型
	commonColumnTypeMap = map[string]dbi.ColumnDataType{

		"CHAR":          dbi.CommonTypeChar, // 字符数据类型
		"VARCHAR":       dbi.CommonTypeVarchar,
		"TEXT":          dbi.CommonTypeText,
		"LONG":          dbi.CommonTypeText,
		"LONGVARCHAR":   dbi.CommonTypeLongtext,
		"IMAGE":         dbi.CommonTypeLongtext,
		"LONGVARBINARY": dbi.CommonTypeLongtext,
		"BLOB":          dbi.CommonTypeBlob,
		"CLOB":          dbi.CommonTypeText,
		"NUMERIC":       dbi.CommonTypeNumber, // 精确数值数据类型
		"DECIMAL":       dbi.CommonTypeNumber,
		"NUMBER":        dbi.CommonTypeNumber,
		"INTEGER":       dbi.CommonTypeInt,
		"INT":           dbi.CommonTypeInt,
		"BIGINT":        dbi.CommonTypeBigint,
		"TINYINT":       dbi.CommonTypeTinyint,
		"BYTE":          dbi.CommonTypeTinyint,
		"SMALLINT":      dbi.CommonTypeSmallint,
		"BIT":           dbi.CommonTypeTinyint,
		"DOUBLE":        dbi.CommonTypeNumber, // 近似数值类型
		"FLOAT":         dbi.CommonTypeNumber,
		"DATE":          dbi.CommonTypeDate, // 一般日期时间数据类型
		"TIME":          dbi.CommonTypeTime,
		"TIMESTAMP":     dbi.CommonTypeTimestamp,
	}

	// 公共数据类型 对应 达梦数据类型
	dmColumnTypeMap = map[dbi.ColumnDataType]string{
		dbi.CommonTypeVarchar:    "VARCHAR",
		dbi.CommonTypeChar:       "CHAR",
		dbi.CommonTypeText:       "TEXT",
		dbi.CommonTypeBlob:       "BLOB",
		dbi.CommonTypeLongblob:   "TEXT",
		dbi.CommonTypeLongtext:   "TEXT",
		dbi.CommonTypeBinary:     "TEXT",
		dbi.CommonTypeMediumblob: "TEXT",
		dbi.CommonTypeMediumtext: "TEXT",
		dbi.CommonTypeVarbinary:  "TEXT",
		dbi.CommonTypeInt:        "INT",
		dbi.CommonTypeSmallint:   "SMALLINT",
		dbi.CommonTypeTinyint:    "TINYINT",
		dbi.CommonTypeNumber:     "NUMBER",
		dbi.CommonTypeBigint:     "BIGINT",
		dbi.CommonTypeDatetime:   "TIMESTAMP",
		dbi.CommonTypeDate:       "DATE",
		dbi.CommonTypeTime:       "DATE",
		dbi.CommonTypeTimestamp:  "TIMESTAMP",
		dbi.CommonTypeEnum:       "TEXT",
		dbi.CommonTypeJSON:       "TEXT",
	}
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
	if dateRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDate
	}
	if timeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeTime
	}
	return dbi.DataTypeString
}

func (dc *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
	str := anyx.ToString(dbColumnValue)
	switch dataType {
	case dbi.DataTypeDateTime: // "2024-01-02T22:08:22.275697+08:00"
		// 尝试用时间格式解析
		res, err := time.Parse(time.DateTime, str)
		if err == nil {
			return str
		}
		res, _ = time.Parse(time.RFC3339, str)
		return res.Format(time.DateTime)
	case dbi.DataTypeDate: // "2024-01-02T00:00:00+08:00"
		// 尝试用时间格式解析
		res, err := time.Parse(time.DateOnly, str)
		if err == nil {
			return str
		}
		res, _ = time.Parse(time.RFC3339, str)
		return res.Format(time.DateOnly)
	case dbi.DataTypeTime: // "0000-01-01T22:08:22.275688+08:00"
		// 尝试用时间格式解析
		res, err := time.Parse(time.TimeOnly, str)
		if err == nil {
			return str
		}
		res, _ = time.Parse(time.RFC3339, str)
		return res.Format(time.TimeOnly)
	}
	return str
}

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
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
		return fmt.Sprintf("'%s'", dc.FormatData(dbColumnValue, dataType))
	}
	return fmt.Sprintf("'%s'", dbColumnValue)
}
