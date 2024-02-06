package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"
	"time"
)

const (
	MSSQL_META_FILE           = "metasql/mssql_meta.sql"
	MSSQL_DBS_KEY             = "MSSQL_DBS"
	MSSQL_DB_SCHEMAS_KEY      = "MSSQL_DB_SCHEMAS"
	MSSQL_TABLE_INFO_KEY      = "MSSQL_TABLE_INFO"
	MSSQL_INDEX_INFO_KEY      = "MSSQL_INDEX_INFO"
	MSSQL_COLUMN_MA_KEY       = "MSSQL_COLUMN_MA"
	MSSQL_TABLE_DETAIL_KEY    = "MSSQL_TABLE_DETAIL"
	MSSQL_TABLE_INDEX_DDL_KEY = "MSSQL_TABLE_INDEX_DDL"
)

type MssqlDialect struct {
	dc *dbi.DbConn
}

func (md *MssqlDialect) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := md.dc.Query("SELECT @@VERSION as version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["version"]),
	}
	return ds, nil
}

func (md *MssqlDialect) GetDbNames() ([]string, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_DBS_KEY))
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, anyx.ConvString(re["dbname"]))
	}

	return databases, nil
}

// 从连接信息中获取数据库和schema信息
func (md *MssqlDialect) currentSchema() string {
	dbName := md.dc.Info.Database
	schema := ""
	arr := strings.Split(dbName, "/")
	if len(arr) == 2 {
		schema = arr[1]
	}
	return schema
}

// 获取表基础元信息, 如表名等
func (md *MssqlDialect) GetTables() ([]dbi.Table, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_TABLE_INFO_KEY), md.currentSchema())

	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    anyx.ConvString(re["tableName"]),
			TableComment: anyx.ConvString(re["tableComment"]),
			CreateTime:   anyx.ConvString(re["createTime"]),
			TableRows:    anyx.ConvInt(re["tableRows"]),
			DataLength:   anyx.ConvInt64(re["dataLength"]),
			IndexLength:  anyx.ConvInt64(re["indexLength"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (md *MssqlDialect) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dbType := md.dc.Info.Type
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbType.RemoveQuote(val))
	}), ",")

	_, res, err := md.dc.Query(fmt.Sprintf(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_COLUMN_MA_KEY), tableName), md.currentSchema())
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		columns = append(columns, dbi.Column{
			TableName:     anyx.ToString(re["TABLE_NAME"]),
			ColumnName:    anyx.ToString(re["COLUMN_NAME"]),
			ColumnType:    anyx.ToString(re["COLUMN_TYPE"]),
			ColumnComment: anyx.ToString(re["COLUMN_COMMENT"]),
			Nullable:      anyx.ToString(re["NULLABLE"]),
			IsPrimaryKey:  anyx.ConvInt(re["IS_PRIMARY_KEY"]) == 1,
			IsIdentity:    anyx.ConvInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: anyx.ToString(re["COLUMN_DEFAULT"]),
			NumScale:      anyx.ToString(re["NUM_SCALE"]),
		})
	}
	return columns, nil
}

// 获取表主键字段名，不存在主键标识则默认第一个字段
func (md *MssqlDialect) GetPrimaryKey(tablename string) (string, error) {
	columns, err := md.GetColumns(tablename)
	if err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", errorx.NewBiz("[%s] 表不存在", tablename)
	}

	for _, v := range columns {
		if v.IsPrimaryKey {
			return v.ColumnName, nil
		}
	}

	return columns[0].ColumnName, nil
}

// 获取表索引信息
func (md *MssqlDialect) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_INDEX_INFO_KEY), md.currentSchema(), tableName)
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    anyx.ConvString(re["indexName"]),
			ColumnName:   anyx.ConvString(re["columnName"]),
			IndexType:    anyx.ConvString(re["indexType"]),
			IndexComment: anyx.ConvString(re["indexComment"]),
			IsUnique:     anyx.ConvInt(re["isUnique"]) == 1,
			SeqInIndex:   anyx.ConvInt(re["seqInIndex"]),
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	result := make([]dbi.Index, 0)
	key := ""
	for _, v := range indexs {
		// 当前的索引名
		in := v.IndexName
		// 过滤掉主键索引,主键索引名为PK__开头的
		if strings.HasPrefix(in, "PK__") {
			continue
		}
		if key == in {
			// 索引字段已根据名称和顺序排序，故取最后一个即可
			i := len(result) - 1
			// 同索引字段以逗号连接
			result[i].ColumnName = result[i].ColumnName + "," + v.ColumnName
		} else {
			key = in
			result = append(result, v)
		}
	}
	return result, nil
}

func (md MssqlDialect) CopyTableDDL(tableName string, newTableName string) (string, error) {
	if newTableName == "" {
		newTableName = tableName
	}

	// 根据列信息生成建表语句
	var builder strings.Builder
	var commentBuilder strings.Builder

	// 查询表名和表注释, 设置表注释
	_, res, err := md.dc.Query(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_TABLE_DETAIL_KEY), md.currentSchema(), tableName)
	if err != nil {
		return "", err
	}
	tableComment := ""
	if len(res) > 0 {
		tableComment = anyx.ToString(res[0]["tableComment"])
		if tableComment != "" {
			// 注释转义单引号
			tableComment = strings.ReplaceAll(tableComment, "'", "\\'")
			commentBuilder.WriteString(fmt.Sprintf("\nEXEC sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE',N'%s';\n", tableComment, md.currentSchema(), newTableName))
		}
	}

	baseTable := fmt.Sprintf("%s.%s", md.dc.Info.Type.QuoteIdentifier(md.currentSchema()), md.dc.Info.Type.QuoteIdentifier(newTableName))

	// 查询列信息
	columns, err := md.GetColumns(tableName)
	if err != nil {
		return "", err
	}

	builder.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", baseTable))
	pks := make([]string, 0)
	for i, v := range columns {
		nullAble := "NULL"
		if v.Nullable == "NO" {
			nullAble = "NOT NULL"
		}
		builder.WriteString(fmt.Sprintf("\t[%s] %s %s", v.ColumnName, v.ColumnType, nullAble))
		if v.IsIdentity {
			builder.WriteString(" IDENTITY(1,11)")
		}
		if v.ColumnDefault != "" {
			builder.WriteString(fmt.Sprintf(" DEFAULT %s", v.ColumnDefault))
		}
		if v.IsPrimaryKey {
			pks = append(pks, fmt.Sprintf("[%s]", v.ColumnName))
		}
		if i < len(columns)-1 {
			builder.WriteString(",")
		}
		builder.WriteString("\n")
	}
	// 设置主键
	if len(pks) > 0 {
		builder.WriteString(fmt.Sprintf("\tCONSTRAINT PK_%s PRIMARY KEY ( %s )", newTableName, strings.Join(pks, ",")))
	}
	builder.WriteString("\n);\n")

	// 设置字段注释
	for _, v := range columns {
		if v.ColumnComment != "" {
			// 注释转义单引号
			v.ColumnComment = strings.ReplaceAll(v.ColumnComment, "'", "\\'")
			commentBuilder.WriteString(fmt.Sprintf("\nEXEC sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE',N'%s', N'COLUMN', N'%s';\n", v.ColumnComment, md.currentSchema(), newTableName, v.ColumnName))
		}
	}

	// 设置索引
	indexs, err := md.GetTableIndex(tableName)
	if err != nil {
		return "", err
	}
	for _, v := range indexs {
		builder.WriteString(fmt.Sprintf("\nCREATE NONCLUSTERED INDEX [%s] ON %s (%s);\n", v.IndexName, baseTable, v.ColumnName))
		// 设置索引注释
		if v.IndexComment != "" {
			// 注释转义单引号
			v.IndexComment = strings.ReplaceAll(v.IndexComment, "'", "\\'")
			commentBuilder.WriteString(fmt.Sprintf("\nEXEC sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE',N'%s', N'INDEX', N'%s';\n", v.IndexComment, md.currentSchema(), newTableName, v.IndexName))
		}
	}
	return builder.String() + commentBuilder.String(), nil
}

// 获取建表ddl
func (md *MssqlDialect) GetTableDDL(tableName string) (string, error) {
	return md.CopyTableDDL(tableName, "")
}

func (md *MssqlDialect) WalkTableRecord(tableName string, walkFn dbi.WalkQueryRowsFunc) error {
	return md.dc.WalkQueryRows(context.Background(), fmt.Sprintf("SELECT * FROM %s", tableName), walkFn)
}

func (md *MssqlDialect) GetSchemas() ([]string, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_DB_SCHEMAS_KEY))
	if err != nil {
		return nil, err
	}

	schemas := make([]string, 0)
	for _, re := range res {
		schemas = append(schemas, anyx.ConvString(re["SCHEMA_NAME"]))
	}
	return schemas, nil
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (md *MssqlDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, fmt.Errorf("该数据库类型不支持数据库备份与恢复: %v", md.dc.Info.Type)
}

func (md *MssqlDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	schema := md.currentSchema()

	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	// 重复占位符字符串n遍
	repeated = strings.Repeat(placeholder+",", len(values))
	// 去除最后一个逗号
	placeholder = strings.TrimSuffix(repeated, ",")

	baseTable := fmt.Sprintf("%s.%s", md.dc.Info.Type.QuoteIdentifier(schema), md.dc.Info.Type.QuoteIdentifier(tableName))

	sqlStr := fmt.Sprintf("insert into %s (%s) values %s", baseTable, strings.Join(columns, ","), placeholder)
	// 执行批量insert sql
	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}
	// 设置允许填充自增列之后，显示指定列名可以插入自增列
	_, _ = md.dc.Exec(fmt.Sprintf("set identity_insert \"%s\" on", tableName))

	exec, err := md.dc.TxExec(tx, sqlStr, args...)

	_, _ = md.dc.Exec(fmt.Sprintf("set identity_insert \"%s\" off", tableName))

	return exec, err
}

func (md *MssqlDialect) GetDataConverter() dbi.DataConverter {
	return new(DataConverter)
}

var (
	// 数字类型
	numberRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`)
	// 日期时间类型
	datetimeRegexp = regexp.MustCompile(`(?i)datetime|timestamp`)
	// 日期类型
	dateRegexp = regexp.MustCompile(`(?i)date`)
	// 时间类型
	timeRegexp = regexp.MustCompile(`(?i)time`)
)

type DataConverter struct {
}

func (dc *DataConverter) GetDataType(dbColumnType string) dbi.DataType {
	if numberRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	// 日期时间类型
	if datetimeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	// 日期类型
	if dateRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDate
	}
	// 时间类型
	if timeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeTime
	}
	return dbi.DataTypeString
}

func (dc *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
	return anyx.ToString(dbColumnValue)
}

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	return dbColumnValue
}

func (md *MssqlDialect) CopyTable(copy *dbi.DbCopyTable) error {

	schema := md.currentSchema()

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
			columns, err := md.GetColumns(copy.TableName)
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
			identityInsertOff := ""
			if hasIdentity {
				identityInsertOn = fmt.Sprintf("SET IDENTITY_INSERT [%s].[%s] ON", schema, newTableName)
				identityInsertOff = fmt.Sprintf("SET IDENTITY_INSERT [%s].[%s] OFF", schema, newTableName)
			}
			_, err = md.dc.Exec(fmt.Sprintf(" %s INSERT INTO [%s].[%s] (%s) SELECT * FROM [%s].[%s] %s", identityInsertOn, schema, newTableName, columnsSql, schema, copy.TableName, identityInsertOff))
			if err != nil {
				logx.Warnf("复制表[%s]数据失败: %s", copy.TableName, err.Error())
			}
		}()
	}

	return err
}
