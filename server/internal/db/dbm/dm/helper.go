package dm

import (
	"encoding/hex"
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"
	"time"

	"gitee.com/chunanyong/dm"
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

	dmStructTypes = map[string]bool{
		"ST_CURVE":           true, // 表示一条曲线，可以是圆弧、抛物线等
		"ST_LINESTRING":      true, // 表示一条或多条连续的线段
		"ST_GEOMCOLLECTION":  true, // 表示一个几何对象集合，可以包含多个不同类型的几何对象
		"ST_GEOMETRY":        true, // 通用几何对象类型，可以表示点、线、面等任何几何形状
		"ST_MULTICURVE":      true, // 表示多个曲线的集合
		"ST_MULTILINESTRING": true, // 表示多个线串的集合
		"ST_MULTIPOINT":      true, // 表示多个点的集合
		"ST_MULTIPOLYGON":    true, // 表示多个多边形的集合
		"ST_MULTISURFACE":    true, // 表示多个表面的集合
		"ST_POINT":           true, // 表示一个点
		"ST_POLYGON":         true, // 表示一个多边形
		"ST_SURFACE":         true, // 表示一个表面，通常是一个多边形
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
	ctype := dmColumnTypeMap[commonColumn.DataType]

	if ctype == "" {
		commonColumn.DataType = "VARCHAR"
		commonColumn.CharMaxLength = 2000
	} else {
		commonColumn.DataType = dbi.ColumnDataType(ctype)
		ch.FixColumn(commonColumn)
	}
}

func (ch *ColumnHelper) FixColumn(column *dbi.Column) {
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

func (dd *ColumnHelper) GetScanDestPtr(qc *dbi.QueryColumn) any {
	if dmStructTypes[strings.ToUpper(qc.Type)] {
		return &dm.DmStruct{}
	}
	return dd.DefaultColumnHelper.GetScanDestPtr(qc)
}

func (dd *ColumnHelper) ConvertScanDestValue(data any, qc *dbi.QueryColumn) any {
	if data == nil {
		return nil
	}

	// 达梦特殊数据类型
	if dmStruct, ok := data.(*dm.DmStruct); ok {
		return ParseDmStruct(dmStruct)
	}

	return dd.DefaultColumnHelper.ConvertScanDestValue(data, qc)
}

func ParseDmStruct(dmStruct *dm.DmStruct) string {
	if !dmStruct.Valid {
		return ""
	}

	name, _ := dmStruct.GetSQLTypeName()
	attributes, _ := dmStruct.GetAttributes()
	arr := make([]string, len(attributes))
	arr = append(arr, name, "(")

	for i, v := range attributes {
		if blb, ok1 := v.(*dm.DmBlob); ok1 {
			if blb.Valid {
				length, _ := blb.GetLength()
				var dest = make([]byte, length)
				_, _ = blb.Read(dest)
				// 2进制转16进制字符串
				hexStr := hex.EncodeToString(dest)
				arr = append(arr, "0x", strings.ToUpper(hexStr))
			}
		} else {
			arr = append(arr, anyx.ToString(v))
		}
		if i < len(attributes)-1 {
			arr = append(arr, ",")
		}
	}

	arr = append(arr, ")")
	return strings.Join(arr, "")
}

type DumpHelper struct {
}

func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) {

}

func (dh *DumpHelper) BeforeInsertSql(quoteSchema string, tableName string) string {
	return fmt.Sprintf("set identity_insert %s on;", tableName)
}

func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) {
	writer.Write([]byte("COMMIT;\n"))
}
