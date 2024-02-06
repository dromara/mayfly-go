package dbi

import (
	"fmt"
	"strings"

	pq "gitee.com/liuzongyang/libpq"
	"github.com/kanzihuang/vitess/go/vt/sqlparser"
)

type DbType string

const (
	DbTypeMysql      DbType = "mysql"
	DbTypeMariadb    DbType = "mariadb"
	DbTypePostgres   DbType = "postgres"
	DbTypeGauss      DbType = "gauss"
	DbTypeDM         DbType = "dm"
	DbTypeOracle     DbType = "oracle"
	DbTypeSqlite     DbType = "sqlite"
	DbTypeMssql      DbType = "mssql"
	DbTypeKingbaseEs DbType = "kingbaseEs"
	DbTypeVastbase   DbType = "vastbase"
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
	case DbTypePostgres, DbTypeGauss, DbTypeKingbaseEs, DbTypeVastbase:
		return quoteIdentifier(name, `"`)
	case DbTypeMssql:
		return fmt.Sprintf("[%s]", name)
	default:
		return quoteIdentifier(name, `"`)
	}
}

func (dbType DbType) RemoveQuote(name string) string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		return removeQuote(name, "`")
	case DbTypePostgres, DbTypeGauss, DbTypeKingbaseEs, DbTypeVastbase:
		return removeQuote(name, `"`)
	default:
		return removeQuote(name, `"`)
	}
}

func (dbType DbType) QuoteLiteral(literal string) string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		literal = strings.ReplaceAll(literal, `\`, `\\`)
		literal = strings.ReplaceAll(literal, `'`, `''`)
		return "'" + literal + "'"
	case DbTypePostgres, DbTypeGauss, DbTypeKingbaseEs, DbTypeVastbase:
		return pq.QuoteLiteral(literal)
	default:
		return pq.QuoteLiteral(literal)
	}
}

func (dbType DbType) MetaDbName() string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		return ""
	case DbTypePostgres, DbTypeGauss:
		return "postgres"
	case DbTypeDM:
		return ""
	case DbTypeKingbaseEs:
		return "security"
	case DbTypeVastbase:
		return "vastbase"
	default:
		return ""
	}
}

func (dbType DbType) Dialect() sqlparser.Dialect {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		return sqlparser.MysqlDialect{}
	case DbTypePostgres, DbTypeGauss, DbTypeKingbaseEs, DbTypeVastbase:
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

// 移除相关引号
func removeQuote(name, quoter string) string {
	return strings.ReplaceAll(name, quoter, "")
}

func (dbType DbType) StmtSetForeignKeyChecks(check bool) string {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		if check {
			return "SET FOREIGN_KEY_CHECKS = 1;\n"
		} else {
			return "SET FOREIGN_KEY_CHECKS = 0;\n"
		}
	case DbTypePostgres, DbTypeGauss, DbTypeKingbaseEs, DbTypeVastbase:
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
	case DbTypePostgres, DbTypeGauss, DbTypeKingbaseEs, DbTypeVastbase:
		// not currently supported postgres
		return ""
	default:
		return ""
	}
}

func (dbType DbType) SupportingBackup() bool {
	switch dbType {
	case DbTypeMysql, DbTypeMariadb:
		return true
	default:
		return false
	}
}
