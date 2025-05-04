package postgres

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/may-fly/cast"
)

const (
	PGSQL_META_FILE      = "metasql/pgsql_meta.sql"
	PGSQL_DB_SCHEMAS     = "PGSQL_DB_SCHEMAS"
	PGSQL_TABLE_INFO_KEY = "PGSQL_TABLE_INFO"
	PGSQL_INDEX_INFO_KEY = "PGSQL_INDEX_INFO"
	PGSQL_COLUMN_MA_KEY  = "PGSQL_COLUMN_MA"
)

type PgsqlMetadata struct {
	dbi.DefaultMetadata

	dc *dbi.DbConn
}

func (pd *PgsqlMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := pd.dc.Query("SELECT version() as server_version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["server_version"]),
	}
	return ds, nil
}

func (pd *PgsqlMetadata) GetDbNames() ([]string, error) {
	_, res, err := pd.dc.Query("SELECT datname AS dbname FROM pg_database WHERE datistemplate = false AND has_database_privilege(datname, 'CONNECT')")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, cast.ToString(re["dbname"]))
	}

	return databases, nil
}

func (pd *PgsqlMetadata) GetTables(tableNames ...string) ([]dbi.Table, error) {
	dialect := pd.dc.GetDialect()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dialect.Quoter().Trim(val))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = pd.dc.Query(sql)
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    re["tableName"].(string),
			TableComment: cast.ToString(re["tableComment"]),
			CreateTime:   cast.ToString(re["createTime"]),
			TableRows:    cast.ToInt(re["tableRows"]),
			DataLength:   cast.ToInt64(re["dataLength"]),
			IndexLength:  cast.ToInt64(re["indexLength"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (pd *PgsqlMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dialect := pd.dc.GetDialect()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dialect.Quoter().Trim(val))
	}), ",")

	_, res, err := pd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		column := dbi.Column{
			TableName:     cast.ToString(re["tableName"]),
			ColumnName:    cast.ToString(re["columnName"]),
			DataType:      cast.ToString(re["dataType"]),
			CharMaxLength: cast.ToInt(re["charMaxLength"]),
			ColumnComment: cast.ToString(re["columnComment"]),
			Nullable:      cast.ToString(re["nullable"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["isPrimaryKey"]) == 1,
			AutoIncrement: cast.ToInt(re["autoIncrement"]) == 1,
			ColumnDefault: cast.ToString(re["columnDefault"]),
			NumPrecision:  cast.ToInt(re["numPrecision"]),
			NumScale:      cast.ToInt(re["numScale"]),
		}

		pd.dc.GetDbDataType(column.DataType).FixColumn(&column)
		FixColumnDefault(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

func (pd *PgsqlMetadata) GetPrimaryKey(tablename string) (string, error) {
	columns, err := pd.GetColumns(tablename)
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
func (pd *PgsqlMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := pd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    cast.ToString(re["indexName"]),
			ColumnName:   cast.ToString(re["columnName"]),
			IndexType:    cast.ToString(re["IndexType"]),
			IndexComment: cast.ToString(re["indexComment"]),
			IsUnique:     cast.ToInt(re["isUnique"]) == 1,
			SeqInIndex:   cast.ToInt(re["seqInIndex"]),
			IsPrimaryKey: cast.ToInt(re["isPrimaryKey"]) == 1,
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	result := make([]dbi.Index, 0)
	key := ""
	for _, v := range indexs {
		// 当前的索引名
		in := v.IndexName
		if key == in {
			// 索引字段已根据名称和顺序排序，故取最后一个即可
			i := len(result) - 1
			// 同索引字段以逗号连接
			if result[i].ColumnName != v.ColumnName {
				result[i].ColumnName = result[i].ColumnName + "," + v.ColumnName
			}
		} else {
			key = in
			result = append(result, v)
		}
	}
	return result, nil
}

// 获取建表ddl
func (pd *PgsqlMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return dbi.GenTableDDL(pd.dc.GetDialect(), pd, tableName, dropBeforeCreate)
}

// 获取pgsql当前连接的库可访问的schemaNames
func (pd *PgsqlMetadata) GetSchemas() ([]string, error) {
	sql := dbi.GetLocalSql(PGSQL_META_FILE, PGSQL_DB_SCHEMAS)
	_, res, err := pd.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, cast.ToString(re["schemaName"]))
	}
	return schemaNames, nil
}

func (pd *PgsqlMetadata) GetDefaultDb() string {
	switch pd.dc.Info.Type {
	case DbTypePostgres, DbTypeGauss:
		return "postgres"
	case DbTypeKingbaseEs:
		return "security"
	case DbTypeVastbase:
		return "vastbase"
	default:
		return ""
	}
}
