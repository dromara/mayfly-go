package mysql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"time"
)

var (
	mysqlQuoter = dbi.Quoter{
		Prefix:     '`',
		Suffix:     '`',
		IsReserved: dbi.AlwaysReserve,
	}
)

var _ dbi.Dialect = (*MysqlDialect)(nil)

type MysqlDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (md *MysqlDialect) CopyTable(copy *dbi.DbCopyTable) error {
	quote := md.Quoter().QuoteIdent
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// 复制表结构创建表
	if _, err := md.dc.Exec(fmt.Sprintf("create table %s like %s", quote(newTableName), quote(tableName))); err != nil {
		return err
	}

	// 复制数据（异步执行，执行失败仅记录日志）
	if copy.CopyData {
		gox.Go(func() {
			if _, err := md.dc.Exec(fmt.Sprintf("insert into %s select * from %s", quote(newTableName), quote(tableName))); err != nil {
				logx.Errorf("mysql copy table [%s] data failed: %s", tableName, err.Error())
			}
		})
	}
	return nil
}

func (md *MysqlDialect) Quoter() dbi.Quoter {
	return mysqlQuoter
}

func (md *MysqlDialect) GetSQLParser() sqlparser.SqlParser {
	return new(mysql.MysqlParser)
}

// GetSQLSplitter mysql切割器：反斜杠转义 + # 行注释
func (md *MysqlDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	return sqlparser.NewMysqlSplitter()
}

func (md *MysqlDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{Dialect: md}
}
