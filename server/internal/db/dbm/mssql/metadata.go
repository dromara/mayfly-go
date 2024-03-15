package mssql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
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

type MssqlMetaData struct {
	dbi.DefaultMetaData

	dc *dbi.DbConn
}

func (md *MssqlMetaData) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := md.dc.Query("SELECT @@VERSION as version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["version"]),
	}
	return ds, nil
}

func (md *MssqlMetaData) GetDbNames() ([]string, error) {
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

// 获取表基础元信息, 如表名等
func (md *MssqlMetaData) GetTables() ([]dbi.Table, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_TABLE_INFO_KEY), md.dc.Info.CurrentSchema())

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
func (md *MssqlMetaData) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.RemoveQuote(md, val))
	}), ",")

	_, res, err := md.dc.Query(fmt.Sprintf(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_COLUMN_MA_KEY), tableName), md.dc.Info.CurrentSchema())
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
func (md *MssqlMetaData) GetPrimaryKey(tablename string) (string, error) {
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

func (md *MssqlMetaData) getTableIndexWithPK(tableName string) ([]dbi.Index, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_INDEX_INFO_KEY), md.dc.Info.CurrentSchema(), tableName)
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
	// 把查询结果以索引名分组，多个索引字段以逗号连接
	result := make([]dbi.Index, 0)
	key := ""
	for _, v := range indexs {
		// 当前的索引名
		in := v.IndexName
		if key == in {
			// 索引字段已根据名称和字段顺序排序，故取最后一个即可
			i := len(result) - 1
			// 同索引字段以逗号连接
			result[i].ColumnName = result[i].ColumnName + "," + v.ColumnName
		} else {
			key = in
			result = append(result, v)
		}
	}
	return indexs, nil
}

// 获取表索引信息
func (md *MssqlMetaData) GetTableIndex(tableName string) ([]dbi.Index, error) {
	indexs, _ := md.getTableIndexWithPK(tableName)
	result := make([]dbi.Index, 0)
	// 过滤掉主键索引,主键索引名为PK__开头的
	for _, v := range indexs {
		in := v.IndexName
		if strings.HasPrefix(in, "PK__") {
			continue
		}
	}
	return result, nil
}

func (md *MssqlMetaData) CopyTableDDL(tableName string, newTableName string) (string, error) {
	if newTableName == "" {
		newTableName = tableName
	}

	// 根据列信息生成建表语句
	var builder strings.Builder
	var commentBuilder strings.Builder

	// 查询表名和表注释, 设置表注释
	_, res, err := md.dc.Query(dbi.GetLocalSql(MSSQL_META_FILE, MSSQL_TABLE_DETAIL_KEY), md.dc.Info.CurrentSchema(), tableName)
	if err != nil {
		return "", err
	}
	tableComment := ""
	if len(res) > 0 {
		tableComment = anyx.ToString(res[0]["tableComment"])
		if tableComment != "" {
			// 注释转义单引号
			tableComment = strings.ReplaceAll(tableComment, "'", "\\'")
			commentBuilder.WriteString(fmt.Sprintf("\nEXEC sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE',N'%s';\n", tableComment, md.dc.Info.CurrentSchema(), newTableName))
		}
	}

	baseTable := fmt.Sprintf("%s.%s", dbi.QuoteIdentifier(md, md.dc.Info.CurrentSchema()), dbi.QuoteIdentifier(md, newTableName))

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
			commentBuilder.WriteString(fmt.Sprintf("\nEXEC sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE',N'%s', N'COLUMN', N'%s';\n", v.ColumnComment, md.dc.Info.CurrentSchema(), newTableName, v.ColumnName))
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
			commentBuilder.WriteString(fmt.Sprintf("\nEXEC sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE',N'%s', N'INDEX', N'%s';\n", v.IndexComment, md.dc.Info.CurrentSchema(), newTableName, v.IndexName))
		}
	}
	return builder.String() + commentBuilder.String(), nil
}

// 获取建表ddl
func (md *MssqlMetaData) GetTableDDL(tableName string) (string, error) {
	return md.CopyTableDDL(tableName, "")
}

func (md *MssqlMetaData) GetSchemas() ([]string, error) {
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

func (md *MssqlMetaData) GetIdentifierQuoteString() string {
	return "["
}

func (md *MssqlMetaData) GetDataConverter() dbi.DataConverter {
	return converter
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

	converter = new(DataConverter)
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
	// 如果dataType是datetime而dbColumnValue是string类型，则需要转换为time.Time类型
	_, ok := dbColumnValue.(string)
	if dataType == dbi.DataTypeDateTime && ok {
		res, _ := time.Parse(time.RFC3339, anyx.ToString(dbColumnValue))
		return res
	}
	if dataType == dbi.DataTypeDate && ok {
		res, _ := time.Parse(time.DateOnly, anyx.ToString(dbColumnValue))
		return res
	}
	if dataType == dbi.DataTypeTime && ok {
		res, _ := time.Parse(time.TimeOnly, anyx.ToString(dbColumnValue))
		return res
	}
	return dbColumnValue
}
