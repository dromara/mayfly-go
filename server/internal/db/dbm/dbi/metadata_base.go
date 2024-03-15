package dbi

import (
	pq "gitee.com/liuzongyang/libpq"
	"github.com/kanzihuang/vitess/go/vt/sqlparser"
)

type BaseMetaData interface {

	// 默认库
	DefaultDb() string

	// 用于引用 SQL 标识符（关键字）的字符串
	GetIdentifierQuoteString() string

	// QuoteLiteral quotes a 'literal' (e.g. a parameter, often used to pass literal
	// to DDL and other statements that do not accept parameters) to be used as part
	// of an SQL statement.  For example:
	//
	//	exp_date := pq.QuoteLiteral("2023-01-05 15:00:00Z")
	//	err := db.Exec(fmt.Sprintf("CREATE ROLE my_user VALID UNTIL %s", exp_date))
	//
	// Any single quotes in name will be escaped. Any backslashes (i.e. "\") will be
	// replaced by two backslashes (i.e. "\\") and the C-style escape identifier
	QuoteLiteral(literal string) string

	SqlParserDialect() sqlparser.Dialect
}

// 默认实现，若需要覆盖，则由各个数据库MetaData实现去覆盖重写
type DefaultMetaData struct {
}

func (dd *DefaultMetaData) DefaultDb() string {
	return ""
}

func (dd *DefaultMetaData) GetIdentifierQuoteString() string {
	return `"`
}

func (dd *DefaultMetaData) QuoteLiteral(literal string) string {
	return pq.QuoteLiteral(literal)
}

func (dd *DefaultMetaData) SqlParserDialect() sqlparser.Dialect {
	return sqlparser.PostgresDialect{}
}
