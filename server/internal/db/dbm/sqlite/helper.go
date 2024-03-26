package sqlite

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
	"regexp"
	"strings"
	"time"
)

var (
	// 数字类型
	numberRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit|real`)
	// 日期时间类型
	datetimeRegexp = regexp.MustCompile(`(?i)datetime`)

	dataTypeRegexp = regexp.MustCompile(`(\w+)\((\d*),?(\d*)\)`)

	dateHelper = new(DataHelper)

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

type DataHelper struct {
}

func (dc *DataHelper) GetDataType(dbColumnType string) dbi.DataType {
	if numberRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	if datetimeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	return dbi.DataTypeString
}

func (dc *DataHelper) FormatData(dbColumnValue any, dataType dbi.DataType) string {
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

func (dc *DataHelper) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	return dbColumnValue
}

func (dc *DataHelper) WrapValue(dbColumnValue any, dataType dbi.DataType) string {
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

type ColumnHelper struct {
}

func (ch *ColumnHelper) ToCommonColumn(dialectColumn *dbi.Column) {
	// 翻译为通用数据库类型
	dataType := dialectColumn.DataType
	t1 := commonColumnTypeMap[string(dataType)]
	if t1 == "" {
		dialectColumn.DataType = dbi.CommonTypeVarchar
		dialectColumn.CharMaxLength = 2000
	} else {
		dialectColumn.DataType = t1
	}
}

func (ch *ColumnHelper) ToColumn(commonColumn *dbi.Column) {
	ctype := sqliteColumnTypeMap[commonColumn.DataType]
	if ctype == "" {
		commonColumn.DataType = "nvarchar"
		commonColumn.CharMaxLength = 2000
	} else {
		ch.FixColumn(commonColumn)
	}
}

func (ch *ColumnHelper) FixColumn(column *dbi.Column) {

}

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

func (db *DumpHelper) BeforeInsert(writer io.Writer, tableName string) {
}
func (db *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) {
}
