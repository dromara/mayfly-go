package mysql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
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

	blobRegexp = regexp.MustCompile(`(?i)blob`)

	//  mysql数据类型 映射 公共数据类型
	commonColumnTypeMap = map[string]dbi.ColumnDataType{
		"bigint":     dbi.CommonTypeBigint,
		"binary":     dbi.CommonTypeBinary,
		"blob":       dbi.CommonTypeBlob,
		"char":       dbi.CommonTypeChar,
		"datetime":   dbi.CommonTypeDatetime,
		"date":       dbi.CommonTypeDate,
		"decimal":    dbi.CommonTypeNumber,
		"double":     dbi.CommonTypeNumber,
		"enum":       dbi.CommonTypeEnum,
		"float":      dbi.CommonTypeNumber,
		"int":        dbi.CommonTypeInt,
		"json":       dbi.CommonTypeJSON,
		"longblob":   dbi.CommonTypeLongblob,
		"longtext":   dbi.CommonTypeLongtext,
		"mediumblob": dbi.CommonTypeBlob,
		"mediumtext": dbi.CommonTypeMediumtext,
		"bit":        dbi.CommonTypeBit,
		"set":        dbi.CommonTypeVarchar,
		"smallint":   dbi.CommonTypeSmallint,
		"text":       dbi.CommonTypeText,
		"time":       dbi.CommonTypeTime,
		"timestamp":  dbi.CommonTypeTimestamp,
		"tinyint":    dbi.CommonTypeTinyint,
		"varbinary":  dbi.CommonTypeVarbinary,
		"varchar":    dbi.CommonTypeVarchar,
	}

	// 公共数据类型 映射 mysql数据类型
	mysqlColumnTypeMap = map[dbi.ColumnDataType]string{
		dbi.CommonTypeVarchar:    "varchar",
		dbi.CommonTypeChar:       "char",
		dbi.CommonTypeText:       "text",
		dbi.CommonTypeBlob:       "blob",
		dbi.CommonTypeLongblob:   "longblob",
		dbi.CommonTypeLongtext:   "longtext",
		dbi.CommonTypeBinary:     "binary",
		dbi.CommonTypeMediumblob: "blob",
		dbi.CommonTypeMediumtext: "mediumtext",
		dbi.CommonTypeVarbinary:  "varbinary",
		dbi.CommonTypeInt:        "int",
		dbi.CommonTypeBit:        "bit",
		dbi.CommonTypeSmallint:   "smallint",
		dbi.CommonTypeTinyint:    "tinyint",
		dbi.CommonTypeNumber:     "decimal",
		dbi.CommonTypeBigint:     "bigint",
		dbi.CommonTypeDatetime:   "datetime",
		dbi.CommonTypeDate:       "date",
		dbi.CommonTypeTime:       "time",
		dbi.CommonTypeTimestamp:  "timestamp",
		dbi.CommonTypeEnum:       "enum",
		dbi.CommonTypeJSON:       "json",
	}
)

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
	// blob类型
	if blobRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeBlob
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
	if ok {
		if dataType == dbi.DataTypeDateTime {
			res, _ := time.Parse(time.DateTime, anyx.ToString(dbColumnValue))
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
		// mysql时间类型无需格式化
		return fmt.Sprintf("'%s'", dbColumnValue)
	case dbi.DataTypeBlob:
		return fmt.Sprintf("unhex('%s')", dbColumnValue)
	}
	return fmt.Sprintf("'%s'", dbColumnValue)
}

type ColumnHelper struct {
}

func (ch *ColumnHelper) ToCommonColumn(dialectColumn *dbi.Column) {
	dataType := dialectColumn.DataType

	t1 := commonColumnTypeMap[string(dataType)]
	commonColumnType := dbi.CommonTypeVarchar

	if t1 != "" {
		commonColumnType = t1
	}

	dialectColumn.DataType = commonColumnType
}

func (ch *ColumnHelper) ToColumn(column *dbi.Column) {
	ctype := mysqlColumnTypeMap[column.DataType]
	if ctype == "" {
		column.DataType = "varchar"
		column.CharMaxLength = 1000
	} else {
		column.DataType = dbi.ColumnDataType(ctype)
		ch.FixColumn(column)
	}
}

func (ch *ColumnHelper) FixColumn(column *dbi.Column) {
	// 如果是int整型，删除精度
	if strings.Contains(strings.ToLower(string(column.DataType)), "int") {
		column.NumScale = 0
		column.CharMaxLength = 0
	} else
	// 如果是text，删除长度
	if strings.Contains(strings.ToLower(string(column.DataType)), "text") {
		column.CharMaxLength = 0
		column.NumPrecision = 0
	}
}
