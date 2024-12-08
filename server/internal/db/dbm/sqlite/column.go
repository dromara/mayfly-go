package sqlite

import "mayfly-go/internal/db/dbm/dbi"

var (
	Integer = dbi.NewDbDataType("integer", dbi.DTInt64).WithCT(dbi.CTInt8)
	Real    = dbi.NewDbDataType("real", dbi.DTNumeric).WithCT(dbi.CTNumeric)
	Text    = dbi.NewDbDataType("text", dbi.DTString).WithCT(dbi.CTText)
	Blob    = dbi.NewDbDataType("blob", dbi.DTBytes).WithCT(dbi.CTBlob)

	DateTime = dbi.NewDbDataType("datetime", dbi.DTDateTime).WithCT(dbi.CTDateTime)
	Date     = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate)
	Time     = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime)
)
