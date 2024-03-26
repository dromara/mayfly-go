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

	msMetadata := md.dc.GetMetaData()
	schema := md.dc.Info.CurrentSchema()
	ignoreDupSql := ""
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		// ALTER TABLE dbo.TEST ADD CONSTRAINT uniqueRows UNIQUE (ColA, ColB, ColC, ColD) WITH (IGNORE_DUP_KEY = ON)
		indexs, _ := msMetadata.MetaData.(*MssqlMetaData).getTableIndexWithPK(tableName)
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

	baseTable := fmt.Sprintf("%s.%s", msMetadata.QuoteIdentifier(schema), msMetadata.QuoteIdentifier(tableName))

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
	msMetadata := md.dc.GetMetaData()
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
			name := msMetadata.QuoteIdentifier(col.ColumnName)
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
		if !collx.ArrayContains(identityCols, msMetadata.RemoveQuote(column)) {
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

	sqlTemp := "MERGE INTO " + msMetadata.QuoteIdentifier(schema) + "." + msMetadata.QuoteIdentifier(tableName) + " T1 USING (" + t2 + ") T2 ON " + strings.Join(caseSqls, " AND ")
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
	msMetadata := md.dc.GetMetaData().MetaData.(*MssqlMetaData)
	schema := md.dc.Info.CurrentSchema()

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := copy.TableName + "_copy_" + time.Now().Format("20060102150405")

	// 复制建表语句
	ddl, err := msMetadata.CopyTableDDL(copy.TableName, newTableName)
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

func (md *MssqlDialect) CreateTable(columns []dbi.Column, tableInfo dbi.Table, dropOldTable bool) (int, error) {
	sqlArr := md.dc.GetMetaData().GenerateTableDDL(columns, tableInfo, dropOldTable)
	_, err := md.dc.Exec(strings.Join(sqlArr, ";"))
	return len(sqlArr), err
}

func (md *MssqlDialect) CreateIndex(tableInfo dbi.Table, indexs []dbi.Index) error {
	sqlArr := md.dc.GetMetaData().GenerateIndexDDL(indexs, tableInfo)
	_, err := md.dc.Exec(strings.Join(sqlArr, ";"))
	return err
}
