package dbm

import (
	"fmt"
	"strings"

	"github.com/kanzihuang/vitess/go/vt/sqlparser"
	"github.com/lib/pq"
)

type DbType string

const (
	DbTypeMysql    DbType = "mysql"
	DbTypePostgres DbType = "postgres"
)

func (dbType DbType) MetaDbName() string {
	switch dbType {
	case DbTypeMysql:
		return "information_schema"
	case DbTypePostgres:
		return "postgres"
	default:
		panic(fmt.Sprintf("invalid database type: %s", dbType))
	}
}

func (dbType DbType) QuoteIdentifier(name string) string {
	switch dbType {
	case DbTypeMysql:
		return quoteIdentifier(name, "`")
	case DbTypePostgres:
		return pq.QuoteIdentifier(name)
	default:
		panic(fmt.Sprintf("invalid database type: %s", dbType))
	}
}

func (dbType DbType) QuoteLiteral(literal string) string {
	switch dbType {
	case DbTypeMysql:
		literal = strings.ReplaceAll(literal, `\`, `\\`)
		literal = strings.ReplaceAll(literal, `'`, `''`)
		return "'" + literal + "'"
	case DbTypePostgres:
		return pq.QuoteLiteral(literal)
	default:
		panic(fmt.Sprintf("invalid database type: %s", dbType))
	}
}

func (dbType DbType) StmtSelectDbName() string {
	switch dbType {
	case DbTypeMysql:
		return "SELECT SCHEMA_NAME AS dbname FROM SCHEMATA"
	case DbTypePostgres:
		return "SELECT datname AS dbname FROM pg_database"
	default:
		panic(fmt.Sprintf("invalid database type: %s", dbType))
	}
}

func (dbType DbType) Dialect() sqlparser.Dialect {
	switch dbType {
	case DbTypeMysql:
		return sqlparser.MysqlDialect{}
	case DbTypePostgres:
		return sqlparser.PostgresDialect{}
	default:
		panic(fmt.Sprintf("invalid database type: %s", dbType))
	}
}

// QuoteIdentifier quotes an "identifier" (e.g. a table or a column name) to be
// used as part of an SQL statement.  For example:
//
//	tblname := "my_table"
//	data := "my_data"
//	quoted := pq.QuoteIdentifier(tblname)
//	err := db.Exec(fmt.Sprintf("INSERT INTO %s VALUES ($1)", quoted), data)
//
// Any double quotes in name will be escaped.  The quoted identifier will be
// case sensitive when used in a query.  If the input string contains a zero
// byte, the result will be truncated immediately before it.
func quoteIdentifier(name, quoter string) string {
	end := strings.IndexRune(name, 0)
	if end > -1 {
		name = name[:end]
	}
	return quoter + strings.Replace(name, quoter, quoter+quoter, -1) + quoter
}

func (dbType DbType) StmtSetForeignKeyChecks(check bool) string {
	switch dbType {
	case DbTypeMysql:
		if check {
			return "SET FOREIGN_KEY_CHECKS = 1;\n"
		} else {
			return "SET FOREIGN_KEY_CHECKS = 0;\n"
		}
	case DbTypePostgres:
		// not currently supported postgres
		return ""
	default:
		panic(fmt.Sprintf("invalid database type: %s", dbType))
	}
}

func (dbType DbType) StmtUseDatabase(dbName string) string {
	switch dbType {
	case DbTypeMysql:
		return fmt.Sprintf("USE %s;\n", dbType.QuoteIdentifier(dbName))
	case DbTypePostgres:
		// not currently supported postgres
		return ""
	default:
		panic(fmt.Sprintf("invalid database type: %s", dbType))
	}
}
