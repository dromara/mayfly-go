package dm

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/may-fly/cast"
)

const (
	DM_META_FILE            = "metasql/dm_meta.sql"
	DM_DB_SCHEMAS           = "DM_DB_SCHEMAS"
	DM_TABLE_INFO_KEY       = "DM_TABLE_INFO"
	DM_TABLE_INFO_NAME_ONLY = "DM_TABLE_INFO_NAME_ONLY" // 当查询表详情失败时，可能是因为没有系统表查询权限，所以尝试只查表名
	DM_INDEX_INFO_KEY       = "DM_INDEX_INFO"
	DM_COLUMN_MA_KEY        = "DM_COLUMN_MA"
	DM_COLUMN_MA_EX_KEY     = "DM_COLUMN_MA_EX"
)

type DMMetadata struct {
	dbi.DefaultMetadata

	dc *dbi.DbConn
}

func (dd *DMMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := dd.dc.Query("select * from v$instance")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["SVR_VERSION"]),
	}
	return ds, nil
}

func (dd *DMMetadata) GetDbNames() ([]string, error) {
	_, res, err := dd.dc.Query("SELECT name AS DBNAME FROM v$database")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, cast.ToString(re["DBNAME"]))
	}

	return databases, nil
}

func (dd *DMMetadata) GetTables(tableNames ...string) ([]dbi.Table, error) {
	dialect := dd.dc.GetDialect()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dialect.Quoter().Trim(val))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(DM_META_FILE, DM_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = dd.dc.Query(sql)
	if err != nil {
		// 尝试只查表名
		sql, err = stringx.TemplateParse(dbi.GetLocalSql(DM_META_FILE, DM_TABLE_INFO_NAME_ONLY), collx.M{"tableNames": names})
		_, res, err = dd.dc.Query(sql)
		if err != nil {
			return nil, err
		}
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    cast.ToString(re["TABLE_NAME"]),
			TableComment: cast.ToString(re["TABLE_COMMENT"]),
			CreateTime:   cast.ToString(re["CREATE_TIME"]),
			TableRows:    cast.ToInt(re["TABLE_ROWS"]),
			DataLength:   cast.ToInt64(re["DATA_LENGTH"]),
			IndexLength:  cast.ToInt64(re["INDEX_LENGTH"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (dd *DMMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dialect := dd.dc.GetDialect()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dialect.Quoter().Trim(val))
	}), ",")

	_, res, err := dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_COLUMN_MA_KEY), tableName))
	if err != nil {
		_, res, err = dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_COLUMN_MA_EX_KEY), tableName))
		if err != nil {
			return nil, err
		}
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		column := dbi.Column{
			TableName:     cast.ToString(re["TABLE_NAME"]),
			ColumnName:    cast.ToString(re["COLUMN_NAME"]),
			DataType:      anyx.ToString(re["DATA_TYPE"]),
			CharMaxLength: cast.ToInt(re["CHAR_MAX_LENGTH"]),
			ColumnComment: cast.ToString(re["COLUMN_COMMENT"]),
			Nullable:      cast.ToString(re["NULLABLE"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["IS_PRIMARY_KEY"]) == 1,
			AutoIncrement: cast.ToInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: cast.ToString(re["COLUMN_DEFAULT"]),
			NumPrecision:  cast.ToInt(re["NUM_PRECISION"]),
			NumScale:      cast.ToInt(re["NUM_SCALE"]),
		}
		dd.dc.GetDbDataType(column.DataType).FixColumn(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

func (dd *DMMetadata) GetPrimaryKey(tablename string) (string, error) {
	columns, err := dd.GetColumns(tablename)
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
func (dd *DMMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_INDEX_INFO_KEY), tableName))
	if err != nil {
		logx.Error("查询达梦索引信息失败", err)
		return nil, nil
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    cast.ToString(re["INDEX_NAME"]),
			ColumnName:   cast.ToString(re["COLUMN_NAME"]),
			IndexType:    cast.ToString(re["INDEX_TYPE"]),
			IndexComment: cast.ToString(re["INDEX_COMMENT"]),
			IsUnique:     cast.ToInt(re["IS_UNIQUE"]) == 1,
			SeqInIndex:   cast.ToInt(re["SEQ_IN_INDEX"]),
			IsPrimaryKey: false,
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
			result[i].ColumnName = result[i].ColumnName + "," + v.ColumnName
		} else {
			key = in
			result = append(result, v)
		}
	}
	return result, nil
}

// 获取建表ddl
func (dd *DMMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return dbi.GenTableDDL(dd.dc.GetDialect(), dd, tableName, dropBeforeCreate)
}

// 获取DM当前连接的库可访问的schemaNames
func (dd *DMMetadata) GetSchemas() ([]string, error) {
	sql := dbi.GetLocalSql(DM_META_FILE, DM_DB_SCHEMAS)
	_, res, err := dd.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, cast.ToString(re["SCHEMA_NAME"]))
	}
	return schemaNames, nil
}
