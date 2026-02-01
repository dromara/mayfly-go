package dm

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/utils/stringx"
	"strings"
	"time"
)

type DMDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (dd *DMDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName
	metadata := dd.dc.GetMetadata()
	ddl, err := metadata.GetTableDDL(tableName, false)
	if err != nil {
		return err
	}
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// 替换新表名
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("\"%s\"", strings.ToUpper(tableName)), fmt.Sprintf("\"%s\"", strings.ToUpper(newTableName)))
	// 去除空格换行
	ddl = stringx.TrimSpaceAndBr(ddl)
	// sqls, err := sqlparser.SplitStatementToPieces(ddl, sqlparser.WithDialect(dd.dc.GetMetaData().GetSqlParserDialect()))
	sqls := strings.Split(ddl, ";")
	for _, sql := range sqls {
		_, _ = dd.dc.Exec(sql)
	}

	// 复制数据
	if copy.CopyData {
		gox.Go(func() {
			// 设置允许填充自增列之后，显示指定列名可以插入自增列\
			identityInsert := fmt.Sprintf("set identity_insert \"%s\" on", newTableName)
			// 获取列名
			columns, _ := metadata.GetColumns(tableName)
			columnArr := make([]string, 0)
			for _, column := range columns {
				columnArr = append(columnArr, fmt.Sprintf("\"%s\"", column.ColumnName))
			}
			columnStr := strings.Join(columnArr, ",")
			// 插入新数据并显示指定列
			_, _ = dd.dc.Exec(fmt.Sprintf("%s insert into \"%s\" (%s) select %s from \"%s\"", identityInsert, newTableName, columnStr, columnStr, tableName))
		})
	}
	return err
}

func (dd *DMDialect) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}

func (sd *DMDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{
		Dialect:  sd,
		Metadata: sd.dc.GetMetadata(),
	}
}
