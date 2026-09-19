package oracle

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
)

var (
	DTOracleDate = dbi.DTDateTime.Copy().WithSQLValue(func(val any) string {
		// oracle date型需要用函数包裹：to_date('%s', 'yyyy-mm-dd hh24:mi:ss')
		return fmt.Sprintf("to_date('%s', 'yyyy-mm-dd hh24:mi:ss')", val)
	})
)

var (
	CHAR      = dbi.NewDbDataType("CHAR", dbi.DTString).WithCategory(dbi.TCChar)
	NCHAR     = dbi.NewDbDataType("NCHAR", dbi.DTString).WithCategory(dbi.TCChar)
	VARCHAR2  = dbi.NewDbDataType("VARCHAR2", dbi.DTString).WithCategory(dbi.TCVarchar)
	NVARCHAR2 = dbi.NewDbDataType("NVARCHAR2", dbi.DTString).WithCategory(dbi.TCVarchar)

	TEXT          = dbi.NewDbDataType("TEXT", dbi.DTString).WithCategory(dbi.TCText)
	LONG          = dbi.NewDbDataType("LONG", dbi.DTString).WithCategory(dbi.TCText)
	LONGVARCHAR   = dbi.NewDbDataType("LONGVARCHAR", dbi.DTString).WithCategory(dbi.TCLongtext)
	IMAGE         = dbi.NewDbDataType("IMAGE", dbi.DTString).WithCategory(dbi.TCLongtext)
	LONGVARBINARY = dbi.NewDbDataType("LONGVARBINARY", dbi.DTString).WithCategory(dbi.TCLongtext)
	CLOB          = dbi.NewDbDataType("CLOB", dbi.DTString).WithCategory(dbi.TCLongtext)

	BLOB = dbi.NewDbDataType("BLOB", dbi.DTBytes).WithCategory(dbi.TCBlob)

	DECIMAL = dbi.NewDbDataType("DECIMAL", dbi.DTDecimal).WithCategory(dbi.TCDecimal)
	// Oracle的NUMBER是任意精度精确数值（非二进制浮点），必须归CTDecimal：若归CTNumeric，
	// 目标为MySQL时会被映射为double而静默丢失精度（NUMBER(20,4)的20位有效数字转double只剩~15位）
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

	TIME      = dbi.NewDbDataType("TIME", DTOracleDate).WithCategory(dbi.TCTime)
	DATE      = dbi.NewDbDataType("DATE", DTOracleDate).WithCategory(dbi.TCDate)
	TIMESTAMP = dbi.NewDbDataType("TIMESTAMP", DTOracleDate).WithCategory(dbi.TCTimestamp)
)
