package mssql

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"
	"time"
)

var (
	// 数字类型
	numberRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`)
	// 日期时间类型
	datetimeRegexp = regexp.MustCompile(`(?i)datetime|timestamp`)
	// 日期类型
	dateRegexp = regexp.MustCompile(`(?i)date`)
	// 时间类型
	timeRegexp = regexp.MustCompile(`(?i)time`)

	// mssql数据类型 对应 公共数据类型
	commonColumnTypeMap = map[string]dbi.ColumnDataType{
		"bigint":           dbi.CommonTypeBigint,
		"numeric":          dbi.CommonTypeNumber,
		"bit":              dbi.CommonTypeInt,
		"smallint":         dbi.CommonTypeSmallint,
		"decimal":          dbi.CommonTypeNumber,
		"smallmoney":       dbi.CommonTypeNumber,
		"int":              dbi.CommonTypeInt,
		"tinyint":          dbi.CommonTypeSmallint, // mssql tinyint不支持负数
		"money":            dbi.CommonTypeNumber,
		"float":            dbi.CommonTypeNumber, // 近似数字
		"real":             dbi.CommonTypeVarchar,
		"date":             dbi.CommonTypeDate, // 日期和时间
		"datetimeoffset":   dbi.CommonTypeDatetime,
		"datetime2":        dbi.CommonTypeDatetime,
		"smalldatetime":    dbi.CommonTypeDatetime,
		"datetime":         dbi.CommonTypeDatetime,
		"time":             dbi.CommonTypeTime,
		"char":             dbi.CommonTypeChar, // 字符串
		"varchar":          dbi.CommonTypeVarchar,
		"text":             dbi.CommonTypeText,
		"nchar":            dbi.CommonTypeChar,
		"nvarchar":         dbi.CommonTypeVarchar,
		"ntext":            dbi.CommonTypeText,
		"binary":           dbi.CommonTypeBinary,
		"varbinary":        dbi.CommonTypeBinary,
		"cursor":           dbi.CommonTypeVarchar, // 其他
		"rowversion":       dbi.CommonTypeVarchar,
		"hierarchyid":      dbi.CommonTypeVarchar,
		"uniqueidentifier": dbi.CommonTypeVarchar,
		"sql_variant":      dbi.CommonTypeVarchar,
		"xml":              dbi.CommonTypeText,
		"table":            dbi.CommonTypeText,
		"geometry":         dbi.CommonTypeText, // 空间几何类型
		"geography":        dbi.CommonTypeText, // 空间地理类型
	}

	// 公共数据类型 对应 mssql数据类型

	mssqlColumnTypeMap = map[dbi.ColumnDataType]string{
		dbi.CommonTypeVarchar:    "nvarchar",
		dbi.CommonTypeChar:       "nchar",
		dbi.CommonTypeText:       "ntext",
		dbi.CommonTypeBlob:       "ntext",
		dbi.CommonTypeLongblob:   "ntext",
		dbi.CommonTypeLongtext:   "ntext",
		dbi.CommonTypeBinary:     "varbinary",
		dbi.CommonTypeMediumblob: "ntext",
		dbi.CommonTypeMediumtext: "ntext",
		dbi.CommonTypeVarbinary:  "varbinary",
		dbi.CommonTypeInt:        "int",
		dbi.CommonTypeSmallint:   "smallint",
		dbi.CommonTypeTinyint:    "smallint",
		dbi.CommonTypeNumber:     "decimal",
		dbi.CommonTypeBigint:     "bigint",
		dbi.CommonTypeDatetime:   "datetime2",
		dbi.CommonTypeDate:       "date",
		dbi.CommonTypeTime:       "time",
		dbi.CommonTypeTimestamp:  "timestamp",
		dbi.CommonTypeEnum:       "nvarchar",
		dbi.CommonTypeJSON:       "nvarchar",
	}
	dataHelper   = &DataHelper{}
	columnHelper = &ColumnHelper{}
)

func GetDataHelper() *DataHelper {
	return dataHelper
}

type DataHelper struct {
}

func (dc *DataHelper) GetDataType(dbColumnType string) dbi.DataType {
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

func (dc *DataHelper) FormatData(dbColumnValue any, dataType dbi.DataType) string {
	// 如果dataType是datetime而dbColumnValue是string类型，则需要根据类型格式化
	str, ok := dbColumnValue.(string)
	if dataType == dbi.DataTypeDateTime && ok {
		// 尝试用时间格式解析
		res, err := time.Parse(time.DateTime, str)
		if err == nil {
			return str
		}
		res, _ = time.Parse(time.RFC3339, str)
		return res.Format(time.DateTime)
	}
	if dataType == dbi.DataTypeDate && ok {
		// 尝试用时间格式解析
		res, _ := time.Parse(time.DateOnly, str)
		return res.Format(time.DateOnly)
	}
	if dataType == dbi.DataTypeTime && ok {
		res, _ := time.Parse(time.TimeOnly, str)
		return res.Format(time.TimeOnly)
	}
	return anyx.ToString(dbColumnValue)
}

func (dc *DataHelper) ParseData(dbColumnValue any, dataType dbi.DataType) any {
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
	dbi.DefaultColumnHelper
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
	ctype := mssqlColumnTypeMap[commonColumn.DataType]

	if ctype == "" {
		commonColumn.DataType = "varchar"
		commonColumn.CharMaxLength = 2000
	} else {
		commonColumn.DataType = dbi.ColumnDataType(ctype)
		ch.FixColumn(commonColumn)
		// 修复数据库迁移字段长度
		dataType := string(commonColumn.DataType)
		if collx.ArrayAnyMatches([]string{"nvarchar", "nchar"}, dataType) {
			commonColumn.CharMaxLength = commonColumn.CharMaxLength * 2
		}

		if collx.ArrayAnyMatches([]string{"char"}, dataType) {
			// char最大长度4000
			if commonColumn.CharMaxLength >= 4000 {
				commonColumn.DataType = "ntext"
				commonColumn.CharMaxLength = 0
			}
		}
	}
}

func (ch *ColumnHelper) FixColumn(column *dbi.Column) {
	dataType := strings.ToLower(string(column.DataType))

	if collx.ArrayAnyMatches([]string{"date", "time"}, dataType) {
		// 如果是datetime，精度取NumScale字段
		column.CharMaxLength = column.NumScale
	} else if collx.ArrayAnyMatches([]string{"int", "bit", "real", "text", "xml"}, dataType) {
		// 不显示长度的类型
		column.NumPrecision = 0
		column.CharMaxLength = 0
	} else if collx.ArrayAnyMatches([]string{"numeric", "decimal", "float"}, dataType) {
		// 如果是num，长度取精度和小数位数
		column.CharMaxLength = 0
	} else if collx.ArrayAnyMatches([]string{"nvarchar", "nchar"}, dataType) {
		// 如果是nvarchar，可视长度减半
		column.CharMaxLength = column.CharMaxLength / 2
	}

	if collx.ArrayAnyMatches([]string{"char"}, dataType) {
		// char最大长度4000
		if column.CharMaxLength >= 4000 {
			column.DataType = "ntext"
			column.CharMaxLength = 0
		}
	}

}

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

// mssql 在insert语句前后不能识别begin和commit语句
func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) {
}

// mssql 在insert语句前后不能识别begin和commit语句
func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) {
}

func (dh *DumpHelper) BeforeInsertSql(quoteSchema string, tableName string) string {
	return fmt.Sprintf("set identity_insert %s.%s on ", quoteSchema, tableName)
}
