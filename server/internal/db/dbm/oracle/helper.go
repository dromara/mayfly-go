package oracle

import (
	"fmt"
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
	numberTypeRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`)
	dateTimeReg      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	dateTimeIsoReg   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*$`)

	// 日期时间类型
	datetimeTypeRegexp = regexp.MustCompile(`(?i)date|timestamp`)

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

	dataHelper   = &DataHelper{}
	columnHelper = &ColumnHelper{}
)

func GetDataHelper() *DataHelper {
	return dataHelper
}

type DataHelper struct {
}

func (dc *DataHelper) GetDataType(dbColumnType string) dbi.DataType {
	if numberTypeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	// 日期时间类型
	if datetimeTypeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	return dbi.DataTypeString
}

func (dc *DataHelper) FormatData(dbColumnValue any, dataType dbi.DataType) string {
	str := anyx.ToString(dbColumnValue)
	if dateTimeReg.MatchString(str) || dateTimeIsoReg.MatchString(str) {
		dataType = dbi.DataTypeDateTime
	}
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

func (dc *DataHelper) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	// oracle把日期类型的数据转化为time类型
	if dataType == dbi.DataTypeDateTime {
		res, _ := time.Parse(time.RFC3339, cast.ToString(dbColumnValue))
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
		return fmt.Sprintf("to_date('%s', 'yyyy-mm-dd hh24:mi:ss')", dc.FormatData(dbColumnValue, dataType))
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
		// 如果是number类型，需要根据公共类型加上长度, 如 bigint 需要转换为number(19,0)
		if strings.Contains(string(t1), "NUMBER") {
			dialectColumn.CharMaxLength = 19
		}
	}
}

func (ch *ColumnHelper) ToColumn(commonColumn *dbi.Column) {
	ctype := oracleColumnTypeMap[commonColumn.DataType]
	if ctype == "" {
		commonColumn.DataType = "NVARCHAR2"
		commonColumn.CharMaxLength = 2000
	} else {
		commonColumn.DataType = dbi.ColumnDataType(ctype)
		ch.FixColumn(commonColumn)
	}
}

func (ch *ColumnHelper) FixColumn(column *dbi.Column) {
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
