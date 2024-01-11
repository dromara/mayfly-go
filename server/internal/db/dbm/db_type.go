package dbm

import (
	"fmt"
	"strings"

	pq "gitee.com/liuzongyang/libpq"
	"github.com/kanzihuang/vitess/go/vt/sqlparser"
)

type DbType string

const (
	DbTypeMysql    DbType = "mysql"
	DbTypeMariadb  DbType = "mariadb"
	DbTypePostgres DbType = "postgres"
	DbTypeDM       DbType = "dm"
)

func ToDbType(dbType string) DbType {
	return DbType(dbType)
}

func (dbType DbType) Equal(typ string) bool {
	return ToDbType(typ) == dbType
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
func (dbType DbType) QuoteIdentifier(name string) string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		return quoteIdentifier(name, "`")
	case DbTypePostgres:
		return quoteIdentifier(name, `"`)
	default:
		return quoteIdentifier(name, `"`)
	}
}

func (dbType DbType) QuoteLiteral(literal string) string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		literal = strings.ReplaceAll(literal, `\`, `\\`)
		literal = strings.ReplaceAll(literal, `'`, `''`)
		return "'" + literal + "'"
	case DbTypePostgres:
		return pq.QuoteLiteral(literal)
	default:
		return pq.QuoteLiteral(literal)
	}
}

func (dbType DbType) MetaDbName() string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		return ""
	case DbTypePostgres:
		return "postgres"
	case DbTypeDM:
		return ""
	default:
		return ""
	}
}

func (dbType DbType) Dialect() sqlparser.Dialect {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		return sqlparser.MysqlDialect{}
	case DbTypePostgres:
		return sqlparser.PostgresDialect{}
	default:
		return sqlparser.PostgresDialect{}
	}
}

func quoteIdentifier(name, quoter string) string {
	end := strings.IndexRune(name, 0)
	if end > -1 {
		name = name[:end]
	}
	return quoter + strings.Replace(name, quoter, quoter+quoter, -1) + quoter
}

func (dbType DbType) StmtSetForeignKeyChecks(check bool) string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		if check {
			return "SET FOREIGN_KEY_CHECKS = 1;\n"
		} else {
			return "SET FOREIGN_KEY_CHECKS = 0;\n"
		}
	case DbTypePostgres:
		// not currently supported postgres
		return ""
	default:
		return ""
	}
}

func (dbType DbType) StmtUseDatabase(dbName string) string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		return fmt.Sprintf("USE %s;\n", dbType.QuoteIdentifier(dbName))
	case DbTypePostgres:
		// not currently supported postgres
		return ""
	default:
		return ""
	}
}
