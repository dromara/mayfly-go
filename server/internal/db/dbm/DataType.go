package dbm

type DataType string

const (
	DataTypeString   DataType = "string"
	DataTypeNumber   DataType = "number"
	DataTypeDate     DataType = "date"
	DataTypeTime     DataType = "time"
	DataTypeDateTime DataType = "datetime"
)
