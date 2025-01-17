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
	CHAR          = dbi.NewDbDataType("VARCHAR", dbi.DTString).WithCT(dbi.CTVarchar)
	VARCHAR       = dbi.NewDbDataType("VARCHAR", dbi.DTString).WithCT(dbi.CTVarchar)
	TEXT          = dbi.NewDbDataType("TEXT", dbi.DTString).WithCT(dbi.CTText)
	LONG          = dbi.NewDbDataType("LONG", dbi.DTString).WithCT(dbi.CTText)
	LONGVARCHAR   = dbi.NewDbDataType("LONGVARCHAR", dbi.DTString).WithCT(dbi.CTLongtext)
	IMAGE         = dbi.NewDbDataType("IMAGE", dbi.DTString).WithCT(dbi.CTLongtext)
	LONGVARBINARY = dbi.NewDbDataType("LONGVARBINARY", dbi.DTString).WithCT(dbi.CTLongtext)
	CLOB          = dbi.NewDbDataType("CLOB", dbi.DTString).WithCT(dbi.CTLongtext)

	BLOB = dbi.NewDbDataType("BLOB", dbi.DTBytes).WithCT(dbi.CTBlob)

	NUMERIC  = dbi.NewDbDataType("NUMERIC", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	DECIMAL  = dbi.NewDbDataType("DECIMAL", dbi.DTDecimal).WithCT(dbi.CTDecimal)
	NUMBER   = dbi.NewDbDataType("NUMBER", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	INTEGER  = dbi.NewDbDataType("INTEGER", dbi.DTInt32).WithCT(dbi.CTInt4)
	INT      = dbi.NewDbDataType("INT", dbi.DTInt32).WithCT(dbi.CTInt4)
	BIGINT   = dbi.NewDbDataType("BIGINT", dbi.DTInt64).WithCT(dbi.CTInt8)
	TINYINT  = dbi.NewDbDataType("TINYINT", dbi.DTInt8).WithCT(dbi.CTInt1)
	BYTE     = dbi.NewDbDataType("BYTE", dbi.DTInt8).WithCT(dbi.CTInt1)
	SMALLINT = dbi.NewDbDataType("SMALLINT", dbi.DTInt16).WithCT(dbi.CTInt2)
	BIT      = dbi.NewDbDataType("BIT", dbi.DTBit).WithCT(dbi.CTBit)
	DOUBLE   = dbi.NewDbDataType("DOUBLE", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	FLOAT    = dbi.NewDbDataType("FLOAT", dbi.DTNumeric).WithCT(dbi.CTNumeric)

	TIME      = dbi.NewDbDataType("TIME", dbi.DTTime).WithCT(dbi.CTTime).WithFixColumn(dbi.ClearCharMaxLength)
	DATE      = dbi.NewDbDataType("DATE", dbi.DTDate).WithCT(dbi.CTDate).WithFixColumn(dbi.ClearCharMaxLength)
	DATETIME  = dbi.NewDbDataType("DATETIME", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(dbi.ClearCharMaxLength)
	TIMESTAMP = dbi.NewDbDataType("TIMESTAMP", dbi.DTDateTime).WithCT(dbi.CTTimestamp).WithFixColumn(dbi.ClearCharMaxLength)

	ST_CURVE           = dbi.NewDbDataType("ST_CURVE", DTDmStruct).WithCT(dbi.CTVarchar)           // 表示一条曲线，可以是圆弧、抛物线等
	ST_LINESTRING      = dbi.NewDbDataType("ST_LINESTRING", DTDmStruct).WithCT(dbi.CTVarchar)      // 表示一条或多条连续的线段
	ST_GEOMCOLLECTION  = dbi.NewDbDataType("ST_GEOMCOLLECTION", DTDmStruct).WithCT(dbi.CTVarchar)  // 表示一个几何对象集合，可以包含多个不同类型的几何对象
	ST_GEOMETRY        = dbi.NewDbDataType("ST_GEOMETRY", DTDmStruct).WithCT(dbi.CTVarchar)        // 通用几何对象类型，可以表示点、线、面等任何几何形状
	ST_MULTICURVE      = dbi.NewDbDataType("ST_MULTICURVE", DTDmStruct).WithCT(dbi.CTVarchar)      // 表示多个曲线的集合
	ST_MULTILINESTRING = dbi.NewDbDataType("ST_MULTILINESTRING", DTDmStruct).WithCT(dbi.CTVarchar) // 表示多个线串的集合
	ST_MULTIPOINT      = dbi.NewDbDataType("ST_MULTIPOINT", DTDmStruct).WithCT(dbi.CTVarchar)      // 表示多个点的集合
	ST_MULTIPOLYGON    = dbi.NewDbDataType("ST_MULTIPOLYGON", DTDmStruct).WithCT(dbi.CTVarchar)    // 表示多个曲线的集合
	ST_MULTISURFACE    = dbi.NewDbDataType("ST_MULTISURFACE", DTDmStruct).WithCT(dbi.CTVarchar)    // 表示多个表面的集合
	ST_POINT           = dbi.NewDbDataType("ST_POINT", DTDmStruct).WithCT(dbi.CTVarchar)           // 表示一个点
	ST_POLYGON         = dbi.NewDbDataType("ST_POLYGON", DTDmStruct).WithCT(dbi.CTVarchar)         //表示一个多边形
	ST_SURFACE         = dbi.NewDbDataType("ST_SURFACE", DTDmStruct).WithCT(dbi.CTVarchar)         // 表示一个表面

	TABLES = dbi.NewDbDataType("TABLES", DTDmArray).WithCT(dbi.CTVarchar) // 表示一个数组
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
