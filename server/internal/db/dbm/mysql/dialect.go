package mysql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"time"
)

var (
	mysqlQuoter = dbi.Quoter{
		Prefix:     '`',
		Suffix:     '`',
		IsReserved: dbi.AlwaysReserve,
	}
)

type MysqlDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (md *MysqlDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, nil
	// return NewDbProgramMysql(md.dc), nil
}

func (md *MysqlDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// 复制表结构创建表
	_, err := md.dc.Exec(fmt.Sprintf("create table %s like %s", newTableName, tableName))
	if err != nil {
		return err
	}

	// 复制数据
	if copy.CopyData {
		go func() {
			_, _ = md.dc.Exec(fmt.Sprintf("insert into %s select * from %s", newTableName, tableName))
		}()
	}
	return err
}

func (md *MysqlDialect) Quoter() dbi.Quoter {
	return mysqlQuoter
}

func (md *MysqlDialect) GetSQLParser() sqlparser.SqlParser {
	return new(mysql.MysqlParser)
}

func (md *MysqlDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{Dialect: md}
}
