package sqlite

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"strings"
	"time"
)

var _ dbi.Dialect = (*SqliteDialect)(nil)

type SqliteDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

// GetSQLParser 语法与标准SQL最接近，沿用pgsql解析器。
// 由本方言显式选择解析器（dbi通用层不依赖任何具体方言），复用决策收敛在方言自身
func (sd *SqliteDialect) GetSQLParser() sqlparser.SqlParser {
	return new(pgsql.PgsqlParser)
}

func (sd *SqliteDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")
	ddl, err := sd.dc.GetMetadata().GetTableDDL(tableName, false)
	if err != nil {
		return err
	}
	// 生成建表语句
	// 替换表名
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("CREATE TABLE \"%s\"", tableName), fmt.Sprintf("CREATE TABLE \"%s\"", newTableName))
	// 替换索引名，索引名为按照规范生成的，才能替换，否则未知索引名，无法替换
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("CREATE INDEX \"%s", tableName), fmt.Sprintf("CREATE INDEX \"%s", newTableName))

	// 执行建表语句
	_, err = sd.dc.Exec(ddl)
	if err != nil {
		return err
	}

	// 使用异步线程插入数据（执行失败仅记录日志）
	if copy.CopyData {
		gox.Go(func() {
			// 执行插入语句
			if _, err := sd.dc.Exec(fmt.Sprintf("INSERT INTO \"%s\" SELECT * FROM \"%s\"", newTableName, tableName)); err != nil {
				logx.Errorf("sqlite copy table [%s] data failed: %s", tableName, err.Error())
			}
		})
	}

	return err
}

func (sd *SqliteDialect) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}

// GetSQLSplitter 标准SQL切割器：sqlite字符串中反斜杠为普通字符，
// 若按mysql语义会把 '\' 误判为转义引号导致后续语句被吞入字符串而错切
func (sd *SqliteDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	return sqlparser.NewStdSQLSplitter()
}

func (sd *SqliteDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{
		dialect: sd,
	}
}
