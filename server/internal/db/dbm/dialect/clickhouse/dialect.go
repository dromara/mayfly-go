package clickhouse

import (
	"fmt"
	"time"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
)

var _ dbi.Dialect = (*ClickHouseDialect)(nil)

type ClickHouseDialect struct {
	di *dbi.DbInfo
}

func (cd *ClickHouseDialect) Quoter() dbi.Quoter {
	return dbi.Quoter{
		Prefix:     '`',
		Suffix:     '`',
		IsReserved: dbi.AlwaysReserve,
	}
}

func (cd *ClickHouseDialect) GetDumpTxnWrapper() dbi.DumpTxnWrapper {
	return new(dbi.DefaultDumpTxnWrapper)
}

func (cd *ClickHouseDialect) GetSQLParser() sqlparser.SQLParser {
	return new(pgsql.PgsqlParser)
}

// GetSQLSplitter clickhouse 切割器：# 与 -- 均为行注释（-- 不要求后随空白），反引号为标识符引用符
func (cd *ClickHouseDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	return sqlparser.NewSplitter(tokenizer.ClickhouseConfig)
}

func (cd *ClickHouseDialect) CopyTable(copy *dbi.DbCopyTable) error {
	quote := cd.Quoter().QuoteIdent
	tableName := copy.TableName

	// 生成新表名，为老表名+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// clickhouse不支持create table like，使用create table as复制表结构与引擎（不复制数据）
	if _, err := cd.di.Exec(fmt.Sprintf("create table %s as %s", quote(newTableName), quote(tableName))); err != nil {
		return err
	}

	// 复制数据（异步执行，执行失败仅记录日志）
	if copy.CopyData {
		gox.Go(func() {
			if _, err := cd.di.Exec(fmt.Sprintf("insert into %s select * from %s", quote(newTableName), quote(tableName))); err != nil {
				logx.Errorf("clickhouse copy table [%s] data failed: %s", tableName, err.Error())
			}
		})
	}
	return nil
}

func (cd *ClickHouseDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{DefaultSQLGenerator: dbi.DefaultSQLGenerator{QuoterFn: cd.Quoter}, dialect: cd}
}
