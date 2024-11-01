package mssql

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

type MssqlDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
}

func (md *MssqlDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {

	if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		return md.batchInsertMerge(tx, tableName, columns, values, duplicateStrategy)
	}

	return md.batchInsertSimple(tx, tableName, columns, values, duplicateStrategy)
}

func (md *MssqlDialect) batchInsertSimple(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
	// 把二维数组转为一维数组
	var args []any
	var singleSize int // 一条数据的参数个数
	for i, v := range values {
		if i == 0 {
			singleSize = len(v)
		}
		args = append(args, v...)
	}

	// 判断如果参数超过2000，则分批次执行，mssql允许最大参数为2100，保险起见，这里限制到2000
	if len(args) > 2000 {

		rows := 2000 / singleSize // 每批次最大数据条数
		mp := make(map[any][][]any)

		// 把values拆成多份，每份不能超过rows条
		length := len(values)
		for i := 0; i < length; i += rows {
			if i+rows <= length {
				mp[i] = values[i : i+rows]
			} else {
				mp[i] = values[i:length]
			}
		}

		var count int64
		for _, v := range mp {
			res, err := md.batchInsertSimple(tx, tableName, columns, v, duplicateStrategy)
			if err != nil {
				return count, err
			}
			count += res
		}
		return count, nil
	}

	msMetadata := md.dc.GetMetadata()
	schema := md.dc.Info.CurrentSchema()
	ignoreDupSql := ""
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		// ALTER TABLE dbo.TEST ADD CONSTRAINT uniqueRows UNIQUE (ColA, ColB, ColC, ColD) WITH (IGNORE_DUP_KEY = ON)
		indexs, _ := msMetadata.(*MssqlMetadata).getTableIndexWithPK(tableName)
		// 收集唯一索引涉及到的字段
		uniqueColumns := make([]string, 0)
		for _, index := range indexs {
			if index.IsUnique {
				cols := strings.Split(index.ColumnName, ",")
				for _, col := range cols {
					if !collx.ArrayContains(uniqueColumns, col) {
						uniqueColumns = append(uniqueColumns, col)
					}
				}
			}
		}
		if len(uniqueColumns) > 0 {
			// 设置忽略重复键
			ignoreDupSql = fmt.Sprintf("ALTER TABLE %s.%s ADD CONSTRAINT uniqueRows UNIQUE (%s) WITH (IGNORE_DUP_KEY = {sign})", schema, tableName, strings.Join(uniqueColumns, ","))
			_, _ = md.dc.TxExec(tx, strings.ReplaceAll(ignoreDupSql, "{sign}", "ON"))
		}
	}

	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	// 重复占位符字符串n遍
	repeated = strings.Repeat(placeholder+",", len(values))
	// 去除最后一个逗号
	placeholder = strings.TrimSuffix(repeated, ",")

	baseTable := fmt.Sprintf("%s.%s", md.QuoteIdentifier(schema), md.QuoteIdentifier(tableName))

	sqlStr := fmt.Sprintf("insert into %s (%s) values %s", baseTable, strings.Join(columns, ","), placeholder)
	// 执行批量insert sql

	// 设置允许填充自增列之后，显示指定列名可以插入自增列
	identityInsertOn := fmt.Sprintf("SET IDENTITY_INSERT [%s].[%s] ON", schema, tableName)

	res, err := md.dc.TxExec(tx, fmt.Sprintf("%s %s", identityInsertOn, sqlStr), args...)

	// 执行完之后，设置忽略重复键
	if ignoreDupSql != "" {
		_, _ = md.dc.TxExec(tx, strings.ReplaceAll(ignoreDupSql, "{sign}", "OFF"))
	}
	return res, err
}

func (md *MssqlDialect) batchInsertMerge(tx *sql.Tx, tableName string, columns []string, values [][]any, duplicateStrategy int) (int64, error) {
	msMetadata := md.dc.GetMetadata()
	schema := md.dc.Info.CurrentSchema()

	// 收集MERGE 语句的 ON 子句条件
	caseSqls := make([]string, 0)
	pkCols := make([]string, 0)

	// 查询取出自增列字段, merge update不能修改自增列
	identityCols := make([]string, 0)
	cols, err := msMetadata.GetColumns(tableName)
	if err != nil {
		return 0, err
	}
	for _, col := range cols {
		if col.IsIdentity {
			identityCols = append(identityCols, col.ColumnName)
		}
		if col.IsPrimaryKey {
			pkCols = append(pkCols, col.ColumnName)
			name := md.QuoteIdentifier(col.ColumnName)
			caseSqls = append(caseSqls, fmt.Sprintf(" T1.%s = T2.%s ", name, name))
		}
	}
	if len(pkCols) == 0 {
		return md.batchInsertSimple(tx, tableName, columns, values, duplicateStrategy)
	}
	// 重复数据处理策略
	insertVals := make([]string, 0)
	upds := make([]string, 0)
	insertCols := make([]string, 0)
	// 源数据占位sql
	phs := make([]string, 0)
	for _, column := range columns {
		if !collx.ArrayContains(identityCols, md.RemoveQuote(column)) {
			upds = append(upds, fmt.Sprintf("T1.%s = T2.%s", column, column))
		}
		insertCols = append(insertCols, column)
		insertVals = append(insertVals, fmt.Sprintf("T2.%s", column))
		phs = append(phs, fmt.Sprintf("? %s", column))
	}

	// 把二维数组转为一维数组
	var args []any
	tmp := fmt.Sprintf("select %s", strings.Join(phs, ","))
	t2s := make([]string, 0)
	for _, v := range values {
		args = append(args, v...)
		t2s = append(t2s, tmp)
	}
	t2 := strings.Join(t2s, " UNION ALL ")

	sqlTemp := "MERGE INTO " + md.QuoteIdentifier(schema) + "." + md.QuoteIdentifier(tableName) + " T1 USING (" + t2 + ") T2 ON " + strings.Join(caseSqls, " AND ")
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") VALUES (" + strings.Join(insertVals, ",") + ") "
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(upds, ",")

	// 设置允许填充自增列之后，显示指定列名可以插入自增列
	identityInsertOn := fmt.Sprintf("SET IDENTITY_INSERT [%s].[%s] ON", schema, tableName)

	// 执行merge sql,必须要以分号结尾
	res, err := md.dc.TxExec(tx, fmt.Sprintf("%s %s;", identityInsertOn, sqlTemp), args...)

	if err != nil {
		logx.Errorf("执行sql失败：%s, sql: [ %s ]", err.Error(), sqlTemp)
	}
	return res, err
}

func (md *MssqlDialect) CopyTable(copy *dbi.DbCopyTable) error {
	msMetadata := md.dc.GetMetadata()
	schema := md.dc.Info.CurrentSchema()

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := copy.TableName + "_copy_" + time.Now().Format("20060102150405")

	// 复制建表语句
	ddl, err := md.CopyTableDDL(copy.TableName, newTableName)
	if err != nil {
		return err
	}

	// 执行建表
	_, err = md.dc.Exec(ddl)
	if err != nil {
		return err
	}
	// 复制数据
	if copy.CopyData {
		go func() {
			// 查询所有的列
			columns, err := msMetadata.GetColumns(copy.TableName)
			if err != nil {
				logx.Warnf("复制表[%s]数据失败: %s", copy.TableName, err.Error())
				return
			}
			// 取出每列名, 需要显示指定列名插入数据
			columnNames := make([]string, 0)
			hasIdentity := false
			for _, v := range columns {
				columnNames = append(columnNames, fmt.Sprintf("[%s]", v.ColumnName))
				if v.IsIdentity {
					hasIdentity = true
				}

			}
			columnsSql := strings.Join(columnNames, ",")

			// 复制数据
			// 设置允许填充自增列之后，显示指定列名可以插入自增列
			identityInsertOn := ""
			if hasIdentity {
				identityInsertOn = fmt.Sprintf("SET IDENTITY_INSERT [%s].[%s] ON", schema, newTableName)
			}
			_, err = md.dc.Exec(fmt.Sprintf(" %s INSERT INTO [%s].[%s] (%s) SELECT * FROM [%s].[%s]", identityInsertOn, schema, newTableName, columnsSql, schema, copy.TableName))
			if err != nil {
				logx.Warnf("复制表[%s]数据失败: %s", copy.TableName, err.Error())
			}
		}()
	}

	return err
}

func (md *MssqlDialect) CopyTableDDL(tableName string, newTableName string) (string, error) {
	if newTableName == "" {
		newTableName = tableName
	}
	metadata := md.dc.GetMetadata()
	// 查询表名和表注释, 设置表注释
	tbs, err := metadata.GetTables(tableName)
	if err != nil || len(tbs) < 1 {
		logx.Errorf("获取表信息失败, %s", tableName)
		return "", err
	}
	tabInfo := &dbi.Table{
		TableName:    newTableName,
		TableComment: tbs[0].TableComment,
	}

	// 查询列信息
	columns, err := metadata.GetColumns(tableName)
	if err != nil {
		logx.Errorf("获取列信息失败, %s", tableName)
		return "", err
	}
	sqlArr := md.GenerateTableDDL(columns, *tabInfo, true)

	// 设置索引
	indexs, err := metadata.GetTableIndex(tableName)
	if err != nil {
		logx.Errorf("获取索引信息失败, %s", tableName)
		return strings.Join(sqlArr, ";"), err
	}
	sqlArr = append(sqlArr, md.GenerateIndexDDL(indexs, *tabInfo)...)
	return strings.Join(sqlArr, ";"), nil
}

// 获取建表ddl
func (md *MssqlDialect) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	tbName := tableInfo.TableName
	schemaName := md.dc.Info.CurrentSchema()

	sqlArr := make([]string, 0)

	// 删除表
	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", md.QuoteIdentifier(schemaName), md.QuoteIdentifier(tbName)))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s.%s (\n", md.QuoteIdentifier(schemaName), md.QuoteIdentifier(tbName))
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, md.QuoteIdentifier(column.ColumnName))
		}
		fields = append(fields, md.genColumnBasicSql(column))
		commentTmp := "EXECUTE sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE', N'%s', N'COLUMN', N'%s'"

		// 防止注释内含有特殊字符串导致sql出错
		if column.ColumnComment != "" {
			comment := md.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf(commentTmp, comment, md.dc.Info.CurrentSchema(), tbName, column.ColumnName))
		}
	}

	// create
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \n PRIMARY KEY CLUSTERED (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	// comment
	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		commentTmp := "EXECUTE sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE', N'%s'"

		tableCommentSql = fmt.Sprintf(commentTmp, md.QuoteEscape(tableInfo.TableComment), md.dc.Info.CurrentSchema(), tbName)
	}

	sqlArr = append(sqlArr, createSql)

	if tableCommentSql != "" {
		sqlArr = append(sqlArr, tableCommentSql)
	}
	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}

	return sqlArr
}

// 获取建索引ddl
func (md *MssqlDialect) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
	tbName := tableInfo.TableName
	sqls := make([]string, 0)
	comments := make([]string, 0)
	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}
		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, name := range cols {
			colNames[i] = md.QuoteIdentifier(name)
		}

		sqls = append(sqls, fmt.Sprintf("create %s NONCLUSTERED index %s on %s.%s(%s)", unique, md.QuoteIdentifier(index.IndexName), md.QuoteIdentifier(md.dc.Info.CurrentSchema()), md.QuoteIdentifier(tbName), strings.Join(colNames, ",")))
		if index.IndexComment != "" {
			comment := md.QuoteEscape(index.IndexComment)
			comments = append(comments, fmt.Sprintf("EXECUTE sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE', N'%s', N'INDEX', N'%s'", comment, md.dc.Info.CurrentSchema(), tbName, index.IndexName))
		}
	}
	if len(comments) > 0 {
		sqls = append(sqls, comments...)
	}

	return sqls
}

func (md *MssqlDialect) GetIdentifierQuoteString() string {
	return "["
}

func (md *MssqlDialect) GetDataHelper() dbi.DataHelper {
	return new(DataHelper)
}

func (md *MssqlDialect) GetColumnHelper() dbi.ColumnHelper {
	return new(ColumnHelper)
}

func (md *MssqlDialect) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}

func (md *MssqlDialect) genColumnBasicSql(column dbi.Column) string {
	colName := md.QuoteIdentifier(column.ColumnName)
	dataType := string(column.DataType)

	incr := ""
	if column.IsIdentity {
		incr = " IDENTITY(1,1)"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, dataType) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) &&
				collx.ArrayAnyMatches([]string{"DATE", "TIME"}, strings.ToUpper(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
		}

		if mark {
			defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s", colName, column.GetColumnType(), incr, nullAble, defVal)
	return columnSql
}
