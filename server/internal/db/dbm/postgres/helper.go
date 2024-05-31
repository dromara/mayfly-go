package postgres

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"
	"time"

	"github.com/may-fly/cast"
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

	// 提取pg默认值， 如：'id'::varchar  提取id  ；  '-1'::integer  提取-1
	defaultValueRegexp = regexp.MustCompile(`'([^']*)'`)

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
	str := fmt.Sprintf("%v", dbColumnValue)
	switch dataType {
	case dbi.DataTypeDateTime: // "2024-01-02T22:16:28.545377+08:00"
		// 尝试用时间格式解析
		res, err := time.Parse(time.DateTime, str)
		if err == nil {
			return str
		}
		res, err = time.Parse(time.RFC3339, str)
		return res.Format(time.DateTime)
	case dbi.DataTypeDate: //  "2024-01-02T00:00:00Z"
		// 尝试用时间格式解析
		res, err := time.Parse(time.DateOnly, str)
		if err == nil {
			return str
		}
		res, _ = time.Parse(time.RFC3339, str)
		return res.Format(time.DateOnly)
	case dbi.DataTypeTime: // "0000-01-01T22:16:28.545075+08:00"
		// 尝试用时间格式解析
		res, err := time.Parse(time.TimeOnly, str)
		if err == nil {
			return str
		}
		res, _ = time.Parse(time.RFC3339, str)
		return res.Format(time.TimeOnly)
	}
	return cast.ToString(dbColumnValue)
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

func (ch *ColumnHelper) ToCommonColumn(column *dbi.Column) {
	// 翻译为通用数据库类型
	dataType := column.DataType
	t1 := commonColumnTypeMap[string(dataType)]
	if t1 == "" {
		column.DataType = dbi.CommonTypeVarchar
		column.CharMaxLength = 2000
	} else {
		column.DataType = t1
	}
}

func (ch *ColumnHelper) ToColumn(commonColumn *dbi.Column) {
	ctype := pgsqlColumnTypeMap[commonColumn.DataType]

	if ctype == "" {
		commonColumn.DataType = "varchar"
		commonColumn.CharMaxLength = 2000
	} else {
		commonColumn.DataType = dbi.ColumnDataType(ctype)

	}
}

func (ch *ColumnHelper) FixColumn(column *dbi.Column) {
	dataType := strings.ToLower(string(column.DataType))
	// 哪些字段可以指定长度
	if !collx.ArrayAnyMatches([]string{"char", "time", "bit", "num", "decimal"}, dataType) {
		column.CharMaxLength = 0
		column.NumPrecision = 0
	} else if strings.Contains(dataType, "char") {
		// 如果类型是文本，长度翻倍
		column.CharMaxLength = column.CharMaxLength * 2
	}
	// 如果默认值带冒号，如：'id'::varchar
	if column.ColumnDefault != "" && strings.Contains(column.ColumnDefault, "::") && !strings.HasPrefix(column.ColumnDefault, "nextval") {
		match := defaultValueRegexp.FindStringSubmatch(column.ColumnDefault)
		if len(match) > 1 {
			column.ColumnDefault = match[1]
		}
	}
}

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) {
	// 设置自增序列当前值
	for _, column := range columns {
		if column.IsIdentity {
			seq := fmt.Sprintf("SELECT setval('%s_%s_seq', (SELECT max(%s) FROM %s));\n", tableName, column.ColumnName, column.ColumnName, tableName)
			writer.Write([]byte(seq))
		}
	}

	writer.Write([]byte("COMMIT;\n"))
}
