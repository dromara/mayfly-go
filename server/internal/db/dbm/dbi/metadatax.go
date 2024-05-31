package dbi

import (
	"fmt"
	"strings"
)

// 包装扩展MetaData，提供所有实现MetaData结构体的公共方法
type MetaDataX struct {
	MetaData
}

func NewMetaDataX(metaData MetaData) *MetaDataX {
	return &MetaDataX{metaData}
}

func (md *MetaDataX) QuoteIdentifier(name string) string {
	return QuoteIdentifier(md, name)
}

func (md *MetaDataX) RemoveQuote(name string) string {
	return RemoveQuote(md, name)
}

// QuoteIdentifier quotes an "identifier" (e.g. a table or a column name) to be
// used as part of an SQL statement.  For example:
//
//	tblname := "my_table"
//	data := "my_data"
//	quoted := quoteIdentifier(tblname, '"')
//	err := db.Exec(fmt.Sprintf("INSERT INTO %s VALUES ($1)", quoted), data)
//
// Any double quotes in name will be escaped.  The quoted identifier will be
// case sensitive when used in a query.  If the input string contains a zero
// byte, the result will be truncated immediately before it.
func QuoteIdentifier(metadata MetaData, name string) string {
	quoter := metadata.GetIdentifierQuoteString()
	// 兼容mssql
	if quoter == "[" {
		return fmt.Sprintf("[%s]", name)
	}

	end := strings.IndexRune(name, 0)
	if end > -1 {
		name = name[:end]
	}
	return quoter + strings.Replace(name, quoter, quoter+quoter, -1) + quoter
}

func RemoveQuote(metadata MetaData, name string) string {
	quoter := metadata.GetIdentifierQuoteString()

	// 兼容mssql
	if quoter == "[" {
		return strings.Trim(name, "[]")
	}

	return strings.ReplaceAll(name, quoter, "")
}
