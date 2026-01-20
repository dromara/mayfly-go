package sqlite

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/gox"
	"strings"
	"time"
)

type SqliteDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
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

	// 使用异步线程插入数据
	if copy.CopyData {
		gox.Go(func() {
			// 执行插入语句
			_, _ = sd.dc.Exec(fmt.Sprintf("INSERT INTO \"%s\" SELECT * FROM \"%s\"", newTableName, tableName))
		})
	}

	return err
}

func (sd *SqliteDialect) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}

func (sd *SqliteDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{
		dialect: sd,
	}
}
