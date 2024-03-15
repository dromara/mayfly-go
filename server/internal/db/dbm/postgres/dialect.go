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
func (pd *PgsqlDialect) pgsqlOnDuplicateStrategySql(duplicateStrategy int, tableName string, columns []string) string {
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
func (pd *PgsqlDialect) gaussOnDuplicateStrategySql(duplicateStrategy int, tableName string, columns []string) string {
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
	return pd.dc.Info.CurrentSchema()
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

func (pd *PgsqlDialect) TransColumns(columns []dbi.Column) []dbi.Column {
	var commonColumns []dbi.Column
	for _, column := range columns {
		// 取出当前数据库类型
		arr := strings.Split(column.ColumnType, "(")
		ctype := arr[0]
		// 翻译为通用数据库类型
		t1 := commonColumnTypeMap[ctype]
		if t1 == "" {
			ctype = "varchar(2000)"
		} else {
			// 回写到列信息
			if len(arr) > 1 {
				ctype = t1 + "(" + arr[1]
			} else {
				ctype = t1
			}
		}
		column.ColumnType = ctype
		commonColumns = append(commonColumns, column)
	}
	return commonColumns
}

func (pd *PgsqlDialect) CreateTable(commonColumns []dbi.Column, tableInfo dbi.Table, dropOldTable bool) (int, error) {
	meta := pd.dc.GetMetaData()
	replacer := strings.NewReplacer(";", "", "'", "")
	if dropOldTable {
		_, _ = pd.dc.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableInfo.TableName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", meta.QuoteIdentifier(tableInfo.TableName))
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)
	// 把通用类型转换为达梦类型
	for _, column := range commonColumns {
		// 取出当前数据库类型
		arr := strings.Split(column.ColumnType, "(")
		ctype := arr[0]
		// 翻译为通用数据库类型
		t1 := pgsqlColumnTypeMap[ctype]
		if t1 == "" {
			ctype = "varchar(2000)"
		} else {
			// 回写到列信息
			if len(arr) > 1 {
				if strings.Contains(strings.ToLower(t1), "int") {
					// 如果是数字，类型后不需要带长度
					ctype = t1
				} else if strings.Contains(strings.ToLower(t1), "char") {
					// 如果是字符串，长度翻倍
					match := bracketsRegexp.FindStringSubmatch(column.ColumnType)
					if len(match) > 1 {
						ctype = fmt.Sprintf("%s(%d)", t1, anyx.ConvInt(match[1])*2)
					} else {
						ctype = t1 + "(1000)"
					}
				} else {
					ctype = t1 + "(" + arr[1]
				}
			} else {
				ctype = t1
			}
		}
		column.ColumnType = ctype

		if column.IsPrimaryKey {
			pks = append(pks, meta.QuoteIdentifier(column.ColumnName))
		}
		fields = append(fields, pd.genColumnBasicSql(column))
		commentTmp := "comment on column %s.%s is '%s'"
		// 防止注释内含有特殊字符串导致sql出错
		comment := replacer.Replace(column.ColumnComment)
		columnComments = append(columnComments, fmt.Sprintf(commentTmp, column.TableName, column.ColumnName, comment))
	}
	createSql += strings.Join(fields, ",")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += ")"
	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		commentTmp := "comment on table %s is '%s'"
		tableCommentSql = fmt.Sprintf(commentTmp, tableInfo.TableName, replacer.Replace(tableInfo.TableComment))
	}

	columnCommentSql := strings.Join(columnComments, ";")

	sqls := make([]string, 0)

	if createSql != "" {
		sqls = append(sqls, createSql)
	}
	if tableCommentSql != "" {
		sqls = append(sqls, tableCommentSql)
	}
	if columnCommentSql != "" {
		sqls = append(sqls, columnCommentSql)
	}

	_, err := pd.dc.Exec(strings.Join(sqls, ";"))

	return 1, err
}

func (pd *PgsqlDialect) genColumnBasicSql(column dbi.Column) string {
	meta := pd.dc.GetMetaData()
	colName := meta.QuoteIdentifier(column.ColumnName)

	// 如果是自增类型，需要转换为serial
	if column.IsIdentity {
		if column.ColumnType == "int4" {
			column.ColumnType = "serial"
		} else if column.ColumnType == "int2" {
			column.ColumnType = "smallserial"
		} else if column.ColumnType == "int8" {
			column.ColumnType = "bigserial"
		} else {
			column.ColumnType = "bigserial"
		}

		return fmt.Sprintf(" %s %s NOT NULL", colName, column.ColumnType)
	}

	nullAble := ""
	if column.Nullable == "NO" {
		nullAble = " NOT NULL"
		// 如果字段不能为空，则设置默认值
		if column.ColumnDefault == "" {
			if collx.ArrayAnyMatches([]string{"char", "text", "lob"}, strings.ToLower(column.ColumnType)) {
				// 文本默认值为空字符串
				column.ColumnDefault = " "
			} else if collx.ArrayAnyMatches([]string{"int", "num"}, strings.ToLower(column.ColumnType)) {
				// 数字默认值为0
				column.ColumnDefault = "0"
			}
		}
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(column.ColumnType)) {
			// 如果是文本类型，则默认值不能带括号
			if collx.ArrayAnyMatches([]string{"char", "text", "lob"}, strings.ToLower(column.ColumnType)) {
				column.ColumnDefault = ""
			}

			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(column.ColumnType)) &&
				collx.ArrayAnyMatches([]string{"DATE", "TIME"}, strings.ToUpper(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
		}
		// 如果数据类型是日期时间，则写死默认值函数
		if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(column.ColumnType)) {
			column.ColumnDefault = "CURRENT_TIMESTAMP"
		}

		if column.ColumnDefault != "" {
			if mark {
				defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
			} else {
				defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
			}
		}
	}

	columnSql := fmt.Sprintf(" %s %s %s %s ", colName, column.ColumnType, nullAble, defVal)
	return columnSql
}

func (pd *PgsqlDialect) CreateIndex(tableInfo dbi.Table, indexs []dbi.Index) error {
	sqls := make([]string, 0)
	comments := make([]string, 0)
	for _, index := range indexs {
		// 通过字段、表名拼接索引名
		columnName := strings.ReplaceAll(index.ColumnName, "-", "")
		columnName = strings.ReplaceAll(columnName, "_", "")
		colName := strings.ReplaceAll(columnName, ",", "_")

		keyType := "normal"
		unique := ""
		if index.IsUnique {
			keyType = "unique"
			unique = "unique"
		}
		indexName := fmt.Sprintf("%s_key_%s_%s", keyType, tableInfo.TableName, colName)

		// 如果索引名存在，先删除索引
		sqls = append(sqls, fmt.Sprintf("drop index if exists %s.%s", pd.currentSchema(), indexName))

		// 创建索引
		sqls = append(sqls, fmt.Sprintf("CREATE %s INDEX %s on %s.%s(%s)", unique, indexName, pd.currentSchema(), tableInfo.TableName, index.ColumnName))
		if index.IndexComment != "" {
			comments = append(comments, fmt.Sprintf("COMMENT ON INDEX %s.%s IS '%s'", pd.currentSchema(), indexName, index.IndexComment))
		}
	}
	_, err := pd.dc.Exec(strings.Join(sqls, ";"))
	// 添加注释
	if len(comments) > 0 {
		_, err = pd.dc.Exec(strings.Join(comments, ";"))
	}
	return err
}

func (pd *PgsqlDialect) UpdateSequence(tableName string, columns []dbi.Column) {
	for _, column := range columns {
		if column.IsIdentity {
			_, _ = pd.dc.Exec(fmt.Sprintf("select setval('%s_%s_seq', (SELECT max(%s) from %s))", tableName, column.ColumnName, column.ColumnName, tableName))
		}
	}
}
