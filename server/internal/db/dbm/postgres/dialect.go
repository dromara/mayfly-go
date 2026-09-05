package postgres

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"time"

	"github.com/spf13/cast"
	"mayfly-go/pkg/errorx"
)

type PgsqlDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (pd *PgsqlDialect) CopyTable(copy *dbi.DbCopyTable) error {
	quote := pd.Quoter().Quote
	tableName := copy.TableName
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")
	// 执行根据旧表创建新表
	if _, err := pd.dc.Exec(fmt.Sprintf("create table %s (like %s)", quote(newTableName), quote(tableName))); err != nil {
		return err
	}

	// 复制数据（异步执行，执行失败仅记录日志）
	if copy.CopyData {
		gox.Go(func() {
			if _, err := pd.dc.Exec(fmt.Sprintf("insert into %s select * from %s", quote(newTableName), quote(tableName))); err != nil {
				logx.Errorf("postgres copy table [%s] data failed: %s", tableName, err.Error())
			}
		})
	}

	// 查询旧表的自增字段名 重新设置新表的序列序列器
	_, res, err := pd.dc.Query("select column_name from information_schema.columns where table_name = $1 and column_default like 'nextval%'", tableName)
	if err != nil {
		return err
	}

	for _, re := range res {
		colName := cast.ToString(re["column_name"])
		if colName != "" {
			// 查询自增列当前最大值
			_, maxRes, err := pd.dc.Query(fmt.Sprintf("select max(%s) max_val from %s", quote(colName), quote(tableName)))
			if err != nil {
				return err
			}
			if len(maxRes) == 0 {
				return errorx.NewBizf("failed to get max value of column [%s] in table [%s]: empty result", colName, tableName)
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
			if _, err := pd.dc.Exec(fmt.Sprintf("CREATE SEQUENCE %s START %d INCREMENT 1", quote(newSeqName), maxVal)); err != nil {
				return err
			}
			// 将新表的自增主键序列与主键列相关联（nextval的参数是字符串字面量，需转义单引号而非标识符引用）
			if _, err := pd.dc.Exec(fmt.Sprintf("alter table %s alter column %s set default nextval('%s')", quote(newTableName), quote(colName), dbi.QuoteEscape(newSeqName))); err != nil {
				return err
			}
		}
	}
	return nil
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

func (pd *PgsqlDialect) GetSQLParser() sqlparser.SqlParser {
	return new(pgsql.PgsqlParser)
}

// GetSQLSplitter pg专属切割器：dollar-quoted字符串（$$/$tag$，函数体DO块内分号）、
// E'...'转义字符串（反斜杠转义）、嵌套块注释，标准切割器会在这三类语法上错切
func (pd *PgsqlDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	return sqlparser.NewPgsqlSplitter()
}
