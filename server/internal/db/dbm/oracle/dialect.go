package oracle

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/oracle"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
	"strings"
	"time"
)

var _ dbi.Dialect = (*OracleDialect)(nil)

type OracleDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (od *OracleDialect) CopyTable(copy *dbi.DbCopyTable) error {
	quote := od.Quoter().QuoteIdent
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := strings.ToUpper(copy.TableName + "_copy_" + time.Now().Format("20060102150405"))
	condition := ""
	if !copy.CopyData {
		condition = " where 1 = 2"
	}
	_, err := od.dc.Exec(fmt.Sprintf("create table %s as select * from %s%s", quote(newTableName), quote(copy.TableName), condition))
	return err
}

func (od *OracleDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{
		Dialect: od,
	}
}

func (od *OracleDialect) GetSQLParser() sqlparser.SqlParser {
	return new(oracle.OracleParser)
}

// GetSQLSplitter oracle切割器：q'[..]' 替代引用字面量 + PL-SQL 过程块感知（反斜杠为普通字符）
func (od *OracleDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	return sqlparser.NewSplitter(tokenizer.OracleConfig)
}
