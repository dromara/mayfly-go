package postgres

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"time"

	"github.com/may-fly/cast"
)

type PgsqlDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (pd *PgsqlDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")
	// 执行根据旧表创建新表
	_, err := pd.dc.Exec(fmt.Sprintf("create table %s (like %s)", newTableName, tableName))
	if err != nil {
		return err
	}

	// 复制数据
	if copy.CopyData {
		go func() {
			_, _ = pd.dc.Exec(fmt.Sprintf("insert into %s select * from %s", newTableName, tableName))
		}()
	}

	// 查询旧表的自增字段名 重新设置新表的序列序列器
	_, res, err := pd.dc.Query(fmt.Sprintf("select column_name from information_schema.columns where table_name = '%s' and column_default like 'nextval%%'", tableName))
	if err != nil {
		return err
	}

	for _, re := range res {
		colName := cast.ToString(re["column_name"])
		if colName != "" {
			// 查询自增列当前最大值
			_, maxRes, err := pd.dc.Query(fmt.Sprintf("select max(%s) max_val from %s", colName, tableName))
			if err != nil {
				return err
			}
			maxVal := cast.ToInt(maxRes[0]["max_val"])
			// 序列起始值为1或当前最大值+1
			if maxVal <= 0 {
				maxVal = 1
			} else {
				maxVal += 1
			}

			// 之所以不用tableName_colName_seq是因为gauss会自动创建同名的序列，且无法修改序列起始值，所以直接使用新序列值
			newSeqName := fmt.Sprintf("%s_%s_copy_seq", newTableName, colName)

			// 创建自增序列，当前最大值为旧表最大值
			_, err = pd.dc.Exec(fmt.Sprintf("CREATE SEQUENCE %s START %d INCREMENT 1", newSeqName, maxVal))
			if err != nil {
				return err
			}
			// 将新表的自增主键序列与主键列相关联
			_, err = pd.dc.Exec(fmt.Sprintf("alter table %s alter column %s set default nextval('%s')", newTableName, colName, newSeqName))
			if err != nil {
				return err
			}
		}
	}
	return err
}

func (pd *PgsqlDialect) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}

func (md *PgsqlDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{
		dialect: md,
		dc:      md.dc,
	}
}
