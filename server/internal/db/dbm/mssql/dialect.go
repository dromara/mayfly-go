package mssql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"strings"
	"time"
)

var (
	mssqlQuoter = dbi.Quoter{
		Prefix:     '[',
		Suffix:     ']',
		IsReserved: dbi.AlwaysReserve,
	}
)

type MssqlDialect struct {
	dbi.DefaultDialect

	dc *dbi.DbConn
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
			defer gox.RecoverPanic()
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
				if v.AutoIncrement {
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
		logx.Errorf("failed to get table, %s", tableName)
		return "", err
	}
	tabInfo := &dbi.Table{
		TableName:    newTableName,
		TableComment: tbs[0].TableComment,
	}

	// 查询列信息
	columns, err := metadata.GetColumns(tableName)
	if err != nil {
		logx.Errorf("failed to get columns, %s", tableName)
		return "", err
	}

	sqlGener := md.GetSQLGenerator()
	sqlArr := sqlGener.GenTableDDL(*tabInfo, columns, true)

	// 设置索引
	indexs, err := metadata.GetTableIndex(tableName)
	if err != nil {
		logx.Errorf("failed to get indexs, %s", tableName)
		return strings.Join(sqlArr, ";"), err
	}
	sqlArr = append(sqlArr, sqlGener.GenIndexDDL(*tabInfo, indexs)...)
	return strings.Join(sqlArr, ";"), nil
}

func (md *MssqlDialect) Quoter() dbi.Quoter {
	return mssqlQuoter
}

func (md *MssqlDialect) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}

func (md *MssqlDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{dc: md.dc}
}
