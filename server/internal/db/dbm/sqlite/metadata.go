package sqlite

import (
	"errors"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"strings"
	"time"
)

const (
	SQLITE_META_FILE      = "metasql/sqlite_meta.sql"
	SQLITE_TABLE_INFO_KEY = "SQLITE_TABLE_INFO"
	SQLITE_INDEX_INFO_KEY = "SQLITE_INDEX_INFO"
)

type SqliteMetaData struct {
	dbi.DefaultMetaData

	dc *dbi.DbConn
}

func (sd *SqliteMetaData) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := sd.dc.Query("SELECT SQLITE_VERSION() as version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["version"]),
	}
	return ds, nil
}

func (sd *SqliteMetaData) GetDbNames() ([]string, error) {
	databases := make([]string, 0)
	_, res, err := sd.dc.Query("PRAGMA database_list")
	if err != nil {
		return nil, err
	}
	for _, re := range res {
		databases = append(databases, anyx.ConvString(re["name"]))
	}

	return databases, nil
}

// 获取表基础元信息, 如表名等
func (sd *SqliteMetaData) GetTables(tableNames ...string) ([]dbi.Table, error) {
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.RemoveQuote(sd, val))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(SQLITE_META_FILE, SQLITE_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = sd.dc.Query(sql)
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    anyx.ConvString(re["tableName"]),
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
func (sd *SqliteMetaData) GetColumns(tableNames ...string) ([]dbi.Column, error) {

	columns := make([]dbi.Column, 0)

	for i := 0; i < len(tableNames); i++ {
		tableName := tableNames[i]
		_, res, err := sd.dc.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			logx.Error("获取数据库表字段结构出错", err.Error())
			continue
		}
		for _, re := range res {
			nullable := "YES"
			if anyx.ConvInt(re["notnull"]) == 1 {
				nullable = "NO"
			}
			// 去掉默认值的引号
			defaultValue := anyx.ConvString(re["dflt_value"])
			if strings.Contains(defaultValue, "'") {
				defaultValue = strings.ReplaceAll(defaultValue, "'", "")
			}
			columns = append(columns, dbi.Column{
				TableName:     tableName,
				ColumnName:    anyx.ConvString(re["name"]),
				ColumnType:    strings.ToLower(anyx.ConvString(re["type"])),
				ColumnComment: "",
				Nullable:      nullable,
				IsPrimaryKey:  anyx.ConvInt(re["pk"]) == 1,
				IsIdentity:    anyx.ConvInt(re["pk"]) == 1,
				ColumnDefault: defaultValue,
				NumScale:      0,
			})
		}
	}
	return columns, nil
}

func (sd *SqliteMetaData) GetPrimaryKey(tableName string) (string, error) {
	_, res, err := sd.dc.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return "", err
	}
	for _, re := range res {
		if anyx.ConvInt(re["pk"]) == 1 {
			return anyx.ConvString(re["name"]), nil
		}
	}

	return "", errors.New("不存在主键")
}

// 解析索引创建语句以获取字段信息
func extractIndexFields(indexSQL string) string {
	// 使用正则表达式提取字段信息
	re := regexp.MustCompile(`\((.*?)\)`)
	match := re.FindStringSubmatch(indexSQL)
	if len(match) > 1 {
		fields := strings.Split(match[1], ",")
		for i, field := range fields {
			// 去除空格
			fields[i] = strings.TrimSpace(field)
		}
		return strings.Join(fields, ",")
	}
	return ""
}

// 获取表索引信息
func (sd *SqliteMetaData) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := sd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(SQLITE_META_FILE, SQLITE_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexSql := anyx.ConvString(re["indexSql"])
		isUnique := strings.Contains(indexSql, "CREATE UNIQUE INDEX")

		indexs = append(indexs, dbi.Index{
			IndexName:    anyx.ConvString(re["indexName"]),
			ColumnName:   extractIndexFields(indexSql),
			IndexType:    anyx.ConvString(re["indexType"]),
			IndexComment: anyx.ConvString(re["indexComment"]),
			IsUnique:     isUnique,
			SeqInIndex:   1,
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	return indexs, nil
}

// 获取建表ddl
func (sd *SqliteMetaData) GetTableDDL(tableName string) (string, error) {
	_, res, err := sd.dc.Query("select sql from sqlite_master WHERE tbl_name=? order by type desc", tableName)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	for _, re := range res {
		builder.WriteString(anyx.ConvString(re["sql"]) + "; \n\n")
	}

	return builder.String(), nil
}

func (sd *SqliteMetaData) GetSchemas() ([]string, error) {
	return nil, nil
}

func (sd *SqliteMetaData) GetDataConverter() dbi.DataConverter {
	return converter
}

var (
	// 数字类型
	numberRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit|real`)
	// 日期时间类型
	datetimeRegexp = regexp.MustCompile(`(?i)datetime`)

	converter = new(DataConverter)

	//  sqlite数据类型 映射 公共数据类型
	commonColumnTypeMap = map[string]dbi.ColumnDataType{
		"int":               dbi.CommonTypeInt,
		"integer":           dbi.CommonTypeInt,
		"tinyint":           dbi.CommonTypeTinyint,
		"smallint":          dbi.CommonTypeSmallint,
		"mediumint":         dbi.CommonTypeSmallint,
		"bigint":            dbi.CommonTypeBigint,
		"int2":              dbi.CommonTypeInt,
		"int8":              dbi.CommonTypeInt,
		"character":         dbi.CommonTypeChar,
		"varchar":           dbi.CommonTypeVarchar,
		"varying character": dbi.CommonTypeVarchar,
		"nchar":             dbi.CommonTypeChar,
		"native character":  dbi.CommonTypeVarchar,
		"nvarchar":          dbi.CommonTypeVarchar,
		"text":              dbi.CommonTypeText,
		"clob":              dbi.CommonTypeBlob,
		"blob":              dbi.CommonTypeBlob,
		"real":              dbi.CommonTypeNumber,
		"double":            dbi.CommonTypeNumber,
		"double precision":  dbi.CommonTypeNumber,
		"float":             dbi.CommonTypeNumber,
		"numeric":           dbi.CommonTypeNumber,
		"decimal":           dbi.CommonTypeNumber,
		"boolean":           dbi.CommonTypeTinyint,
		"date":              dbi.CommonTypeDate,
		"datetime":          dbi.CommonTypeDatetime,
	}

	//  公共数据类型 映射 sqlite数据类型
	sqliteColumnTypeMap = map[dbi.ColumnDataType]string{
		dbi.CommonTypeVarchar:    "nvarchar",
		dbi.CommonTypeChar:       "nchar",
		dbi.CommonTypeText:       "text",
		dbi.CommonTypeBlob:       "blob",
		dbi.CommonTypeLongblob:   "blob",
		dbi.CommonTypeLongtext:   "text",
		dbi.CommonTypeBinary:     "text",
		dbi.CommonTypeMediumblob: "blob",
		dbi.CommonTypeMediumtext: "text",
		dbi.CommonTypeVarbinary:  "text",
		dbi.CommonTypeInt:        "int",
		dbi.CommonTypeSmallint:   "smallint",
		dbi.CommonTypeTinyint:    "tinyint",
		dbi.CommonTypeNumber:     "number",
		dbi.CommonTypeBigint:     "bigint",
		dbi.CommonTypeDatetime:   "datetime",
		dbi.CommonTypeDate:       "date",
		dbi.CommonTypeTime:       "datetime",
		dbi.CommonTypeTimestamp:  "datetime",
		dbi.CommonTypeEnum:       "nvarchar(2000)",
		dbi.CommonTypeJSON:       "nvarchar(2000)",
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
	return dbi.DataTypeString
}

func (dc *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
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

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	return dbColumnValue
}
