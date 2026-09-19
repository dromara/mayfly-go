package dm

import (
	"encoding/hex"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
	"reflect"
	"strings"

	"gitee.com/chunanyong/dm"
)

var (
	// CHAR不能误写为"VARCHAR"：注册表按类型名索引，同名会被静默覆盖导致CHAR类型丢失
	CHAR          = dbi.NewDbDataType("CHAR", dbi.DTString).WithCategory(dbi.TCVarchar)
	VARCHAR       = dbi.NewDbDataType("VARCHAR", dbi.DTString).WithCategory(dbi.TCVarchar)
	TEXT          = dbi.NewDbDataType("TEXT", dbi.DTString).WithCategory(dbi.TCText)
	LONG          = dbi.NewDbDataType("LONG", dbi.DTString).WithCategory(dbi.TCText)
	LONGVARCHAR   = dbi.NewDbDataType("LONGVARCHAR", dbi.DTString).WithCategory(dbi.TCLongtext)
	IMAGE         = dbi.NewDbDataType("IMAGE", dbi.DTString).WithCategory(dbi.TCLongtext)
	LONGVARBINARY = dbi.NewDbDataType("LONGVARBINARY", dbi.DTString).WithCategory(dbi.TCLongtext)
	CLOB          = dbi.NewDbDataType("CLOB", dbi.DTString).WithCategory(dbi.TCLongtext)

	BLOB = dbi.NewDbDataType("BLOB", dbi.DTBytes).WithCategory(dbi.TCBlob)

	// 达梦的NUMERIC/DECIMAL/NUMBER均为精确数值（彼此同义），一律归CTDecimal
	NUMERIC = dbi.NewDbDataType("NUMERIC", dbi.DTNumeric).WithCategory(dbi.TCDecimal)
	DECIMAL = dbi.NewDbDataType("DECIMAL", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	// 达梦的NUMBER（Oracle兼容）是任意精度精确数值，必须归CTDecimal：若归CTNumeric，
	// 目标为MySQL时会被映射为double而静默丢失精度
	NUMBER   = dbi.NewDbDataType("NUMBER", dbi.DTNumeric).WithCategory(dbi.TCDecimal)
	INTEGER  = dbi.NewDbDataType("INTEGER", dbi.DTInt32).WithCategory(dbi.TCInt4)
	INT      = dbi.NewDbDataType("INT", dbi.DTInt32).WithCategory(dbi.TCInt4)
	BIGINT   = dbi.NewDbDataType("BIGINT", dbi.DTInt64).WithCategory(dbi.TCInt8)
	TINYINT  = dbi.NewDbDataType("TINYINT", dbi.DTInt8).WithCategory(dbi.TCInt1)
	BYTE     = dbi.NewDbDataType("BYTE", dbi.DTInt8).WithCategory(dbi.TCInt1)
	SMALLINT = dbi.NewDbDataType("SMALLINT", dbi.DTInt16).WithCategory(dbi.TCInt2)
	BIT      = dbi.NewDbDataType("BIT", dbi.DTBit).WithCategory(dbi.TCBit)
	DOUBLE   = dbi.NewDbDataType("DOUBLE", dbi.DTNumeric).WithCategory(dbi.TCNumeric)
	FLOAT    = dbi.NewDbDataType("FLOAT", dbi.DTNumeric).WithCategory(dbi.TCNumeric)

	TIME      = dbi.NewDbDataType("TIME", dbi.DTTime).WithCategory(dbi.TCTime).WithFixColumn(dbi.ClearCharMaxLength)
	DATE      = dbi.NewDbDataType("DATE", dbi.DTDate).WithCategory(dbi.TCDate).WithFixColumn(dbi.ClearCharMaxLength)
	DATETIME  = dbi.NewDbDataType("DATETIME", dbi.DTDateTime).WithCategory(dbi.TCDateTime).WithFixColumn(dbi.ClearCharMaxLength)
	TIMESTAMP = dbi.NewDbDataType("TIMESTAMP", dbi.DTDateTime).WithCategory(dbi.TCTimestamp).WithFixColumn(dbi.ClearCharMaxLength)

	ST_CURVE           = dbi.NewDbDataType("ST_CURVE", DTDmStruct).WithCategory(dbi.TCVarchar)           // 表示一条曲线，可以是圆弧、抛物线等
	ST_LINESTRING      = dbi.NewDbDataType("ST_LINESTRING", DTDmStruct).WithCategory(dbi.TCVarchar)      // 表示一条或多条连续的线段
	ST_GEOMCOLLECTION  = dbi.NewDbDataType("ST_GEOMCOLLECTION", DTDmStruct).WithCategory(dbi.TCVarchar)  // 表示一个几何对象集合，可以包含多个不同类型的几何对象
	ST_GEOMETRY        = dbi.NewDbDataType("ST_GEOMETRY", DTDmStruct).WithCategory(dbi.TCVarchar)        // 通用几何对象类型，可以表示点、线、面等任何几何形状
	ST_MULTICURVE      = dbi.NewDbDataType("ST_MULTICURVE", DTDmStruct).WithCategory(dbi.TCVarchar)      // 表示多个曲线的集合
	ST_MULTILINESTRING = dbi.NewDbDataType("ST_MULTILINESTRING", DTDmStruct).WithCategory(dbi.TCVarchar) // 表示多个线串的集合
	ST_MULTIPOINT      = dbi.NewDbDataType("ST_MULTIPOINT", DTDmStruct).WithCategory(dbi.TCVarchar)      // 表示多个点的集合
	ST_MULTIPOLYGON    = dbi.NewDbDataType("ST_MULTIPOLYGON", DTDmStruct).WithCategory(dbi.TCVarchar)    // 表示多个曲线的集合
	ST_MULTISURFACE    = dbi.NewDbDataType("ST_MULTISURFACE", DTDmStruct).WithCategory(dbi.TCVarchar)    // 表示多个表面的集合
	ST_POINT           = dbi.NewDbDataType("ST_POINT", DTDmStruct).WithCategory(dbi.TCVarchar)           // 表示一个点
	ST_POLYGON         = dbi.NewDbDataType("ST_POLYGON", DTDmStruct).WithCategory(dbi.TCVarchar)         //表示一个多边形
	ST_SURFACE         = dbi.NewDbDataType("ST_SURFACE", DTDmStruct).WithCategory(dbi.TCVarchar)         // 表示一个表面

	TABLES = dbi.NewDbDataType("TABLES", DTDmArray).WithCategory(dbi.TCVarchar) // 表示一个数组
)

var DTDmStruct = &dbi.DataType{
	Name: "dm_struct",
	Valuer: func() dbi.Valuer {
		return &dmStructValuer{
			DefaultValuer: new(dbi.DefaultValuer[dm.DmStruct]),
		}
	},
	SQLValue: dbi.SQLValueString,
}

type dmStructValuer struct {
	*dbi.DefaultValuer[dm.DmStruct]
}

func (s *dmStructValuer) Value() any {
	if !s.ValuePtr.Valid {
		return ""
	}
	return ParseDmStruct(s.ValuePtr)
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

var DTDmArray = &dbi.DataType{
	Name: "dm_struct",
	Valuer: func() dbi.Valuer {
		return &dmArrayValuer{
			DefaultValuer: new(dbi.DefaultValuer[dm.DmArray]),
		}
	},
	SQLValue: dbi.SQLValueString,
}

type dmArrayValuer struct {
	*dbi.DefaultValuer[dm.DmArray]
}

func (s *dmArrayValuer) Value() any {
	if !s.ValuePtr.Valid {
		return ""
	}
	return ParseDmArray(s.ValuePtr)
}

func ParseDmArray(dmArray *dm.DmArray) string {
	if !dmArray.Valid {
		return ""
	}

	name, err := dmArray.GetBaseTypeName()
	if err != nil {
		return err.Error()
	}

	arr, err := dmArray.GetArray()
	if err != nil {
		return err.Error()
	}

	// 获取变量的类型和值
	t := reflect.TypeOf(arr)
	v := reflect.ValueOf(arr)

	// 检查类型是否为数组
	if t.Kind() != reflect.Array && t.Kind() != reflect.Slice {
		return fmt.Sprintf("%s(%s)", name, anyx.ToString(arr))
	}
	// 获取数组的长度
	length := v.Len()
	elements := make([]string, length)

	// 遍历数组并将每个元素转换为字符串
	for i := 0; i < length; i++ {
		element := v.Index(i).Interface()
		elements[i] = fmt.Sprintf("%v", element)
	}

	return fmt.Sprintf("%s(%s)", name, strings.Join(elements, ","))
}
