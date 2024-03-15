package postgres

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

type PgsqlDialect struct {
	dc *dbi.DbConn
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (pd *PgsqlDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, fmt.Errorf("该数据库类型不支持数据库备份与恢复: %v", pd.dc.Info.Type)
}

func (pd *PgsqlDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
	// 执行批量insert sql，跟mysql一样  pg或高斯支持批量insert语法
	// insert into table_name (column1, column2, ...) values (value1, value2, ...), (value1, value2, ...), ...

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}

	// 构建占位符字符串 "($1, $2, $3), ($4, $5, $6), ..." 用于指定参数
	var placeholders []string
	for i := 0; i < len(args); i += len(columns) {
		var placeholder []string
		for j := 0; j < len(columns); j++ {
			placeholder = append(placeholder, fmt.Sprintf("$%d", i+j+1))
		}
		placeholders = append(placeholders, "("+strings.Join(placeholder, ", ")+")")
	}

	// 根据冲突策略生成后缀
	suffix := ""
	if pd.dc.Info.Type == dbi.DbTypeGauss {
		// 高斯db使用ON DUPLICATE KEY UPDATE 语法参考 https://support.huaweicloud.com/distributed-devg-v3-gaussdb/gaussdb-12-0607.html#ZH-CN_TOPIC_0000001633948138
		suffix = pd.gaussOnDuplicateStrategySql(duplicateStrategy, tableName, columns)
	} else {
		// pgsql 默认使用 on conflict 语法参考 http://www.postgres.cn/docs/12/sql-insert.html
		// vastbase语法参考 https://docs.vastdata.com.cn/zh/docs/VastbaseE100Ver3.0.0/doc/SQL%E8%AF%AD%E6%B3%95/INSERT.html
		// kingbase语法参考 https://help.kingbase.com.cn/v8/development/sql-plsql/sql/SQL_Statements_9.html#insert
		suffix = pd.pgsqlOnDuplicateStrategySql(duplicateStrategy, tableName, columns)
	}

	sqlStr := fmt.Sprintf("insert into %s (%s) values %s %s", pd.dc.GetMetaData().QuoteIdentifier(tableName), strings.Join(columns, ","), strings.Join(placeholders, ", "), suffix)
	// 执行批量insert sql

	return pd.dc.TxExec(tx, sqlStr, args...)
}

// pgsql默认唯一键冲突策略
func (pd PgsqlDialect) pgsqlOnDuplicateStrategySql(duplicateStrategy int, tableName string, columns []string) string {
	suffix := ""
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		suffix = " \n on conflict do nothing"
	} else if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		// 生成 on conflict () do update set column1 = excluded.column1, column2 = excluded.column2, ...
		var updateColumns []string
		for _, col := range columns {
			updateColumns = append(updateColumns, fmt.Sprintf("%s = excluded.%s", col, col))
		}
		// 查询唯一键名,拼接冲突sql
		_, keyRes, _ := pd.dc.Query("SELECT constraint_name FROM information_schema.table_constraints WHERE constraint_schema = $1 AND table_name = $2 AND constraint_type in ('PRIMARY KEY', 'UNIQUE') ", pd.currentSchema(), tableName)
		if len(keyRes) > 0 {
			for _, re := range keyRes {
				key := anyx.ToString(re["constraint_name"])
				if key != "" {
					suffix += fmt.Sprintf(" \n on conflict on constraint %s do update set %s \n", key, strings.Join(updateColumns, ", "))
				}
			}
		}
	}
	return suffix
}

// 高斯db唯一键冲突策略,使用ON DUPLICATE KEY UPDATE 参考：https://support.huaweicloud.com/distributed-devg-v3-gaussdb/gaussdb-12-0607.html#ZH-CN_TOPIC_0000001633948138
func (pd PgsqlDialect) gaussOnDuplicateStrategySql(duplicateStrategy int, tableName string, columns []string) string {
	suffix := ""
	metadata := pd.dc.GetMetaData()
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		suffix = " \n ON DUPLICATE KEY UPDATE NOTHING"
	} else if duplicateStrategy == dbi.DuplicateStrategyUpdate {

		// 查出表里的唯一键涉及的字段
		var uniqueColumns []string
		indexs, err := metadata.GetTableIndex(tableName)
		if err == nil {
			for _, index := range indexs {
				if index.IsUnique {
					cols := strings.Split(index.ColumnName, ",")
					for _, col := range cols {
						if !collx.ArrayContains(uniqueColumns, strings.ToLower(col)) {
							uniqueColumns = append(uniqueColumns, strings.ToLower(col))
						}
					}
				}
			}
		}

		suffix = " \n ON DUPLICATE KEY UPDATE "
		for i, col := range columns {
			// ON DUPLICATE KEY UPDATE语句不支持更新唯一键字段，所以得去掉
			if !collx.ArrayContains(uniqueColumns, metadata.RemoveQuote(strings.ToLower(col))) {
				suffix += fmt.Sprintf("%s = excluded.%s", col, col)
				if i < len(columns)-1 {
					suffix += ", "
				}
			}
		}
	}
	return suffix
}

// 从连接信息中获取数据库和schema信息
func (pd *PgsqlDialect) currentSchema() string {
	dbName := pd.dc.Info.Database
	schema := ""
	arr := strings.Split(dbName, "/")
	if len(arr) == 2 {
		schema = arr[1]
	}
	return schema
}

func (pd *PgsqlDialect) IsGauss() bool {
	return strings.Contains(pd.dc.Info.Params, "gauss")
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
		colName := anyx.ConvString(re["column_name"])
		if colName != "" {

			// 查询自增列当前最大值
			_, maxRes, err := pd.dc.Query(fmt.Sprintf("select max(%s) max_val from %s", colName, tableName))
			if err != nil {
				return err
			}
			maxVal := anyx.ConvInt(maxRes[0]["max_val"])
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
