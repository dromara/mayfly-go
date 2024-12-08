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
	CHAR      = dbi.NewDbDataType("CHAR", dbi.DTString).WithCT(dbi.CTChar)
	NCHAR     = dbi.NewDbDataType("NCHAR", dbi.DTString).WithCT(dbi.CTChar)
	VARCHAR2  = dbi.NewDbDataType("VARCHAR2", dbi.DTString).WithCT(dbi.CTVarchar)
	NVARCHAR2 = dbi.NewDbDataType("NVARCHAR2", dbi.DTString).WithCT(dbi.CTVarchar)

	TEXT          = dbi.NewDbDataType("TEXT", dbi.DTString).WithCT(dbi.CTText)
	LONG          = dbi.NewDbDataType("LONG", dbi.DTString).WithCT(dbi.CTText)
	LONGVARCHAR   = dbi.NewDbDataType("LONGVARCHAR", dbi.DTString).WithCT(dbi.CTLongtext)
	IMAGE         = dbi.NewDbDataType("IMAGE", dbi.DTString).WithCT(dbi.CTLongtext)
	LONGVARBINARY = dbi.NewDbDataType("LONGVARBINARY", dbi.DTString).WithCT(dbi.CTLongtext)
	CLOB          = dbi.NewDbDataType("CLOB", dbi.DTString).WithCT(dbi.CTLongtext)

	BLOB = dbi.NewDbDataType("BLOB", dbi.DTBytes).WithCT(dbi.CTBlob)

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

	TIME      = dbi.NewDbDataType("TIME", DTOracleDate).WithCT(dbi.CTTime)
	DATE      = dbi.NewDbDataType("DATE", DTOracleDate).WithCT(dbi.CTDate)
	TIMESTAMP = dbi.NewDbDataType("TIMESTAMP", DTOracleDate).WithCT(dbi.CTTimestamp)
)
