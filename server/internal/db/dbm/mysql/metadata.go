package mysql

import (
	"errors"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/may-fly/cast"
)

const (
	MYSQL_META_FILE      = "metasql/mysql_meta.sql"
	MYSQL_DBS            = "MYSQL_DBS"
	MYSQL_TABLE_INFO_KEY = "MYSQL_TABLE_INFO"
	MYSQL_INDEX_INFO_KEY = "MYSQL_INDEX_INFO"
	MYSQL_COLUMN_MA_KEY  = "MYSQL_COLUMN_MA"
)

type MysqlMetadata struct {
	dbi.DefaultMetadata

	dc *dbi.DbConn
}

func (md *MysqlMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := md.dc.Query("SELECT VERSION() version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["version"]),
	}
	return ds, nil
}

func (md *MysqlMetadata) GetDbNames() ([]string, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MYSQL_META_FILE, MYSQL_DBS))
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, cast.ToString(re["dbname"]))
	}
	return databases, nil
}

func (md *MysqlMetadata) GetTables(tableNames ...string) ([]dbi.Table, error) {
	dialect := md.dc.GetDialect()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dialect.Quoter().Trim(val))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(MYSQL_META_FILE, MYSQL_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = md.dc.Query(sql)
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    cast.ToString(re["tableName"]),
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
func (md *MysqlMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dialect := md.dc.GetDialect()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dialect.Quoter().Trim(val))
	}), ",")

	_, res, err := md.dc.Query(fmt.Sprintf(dbi.GetLocalSql(MYSQL_META_FILE, MYSQL_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {

		column := dbi.Column{
			TableName:     cast.ToString(re["tableName"]),
			ColumnName:    cast.ToString(re["columnName"]),
			DataType:      cast.ToString(re["dataType"]),
			ColumnComment: cast.ToString(re["columnComment"]),
			Nullable:      cast.ToString(re["nullable"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["isPrimaryKey"]) == 1,
			AutoIncrement: cast.ToInt(re["autoIncrement"]) == 1,
			ColumnDefault: cast.ToString(re["columnDefault"]),
			CharMaxLength: cast.ToInt(re["charMaxLength"]),
			NumPrecision:  cast.ToInt(re["numPrecision"]),
			NumScale:      cast.ToInt(re["numScale"]),
		}

		md.dc.GetDbDataType(column.DataType).FixColumn(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

// 获取表主键字段名，不存在主键标识则默认第一个字段
func (md *MysqlMetadata) GetPrimaryKey(tablename string) (string, error) {
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
func (md *MysqlMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MYSQL_META_FILE, MYSQL_INDEX_INFO_KEY), tableName)
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    cast.ToString(re["indexName"]),
			ColumnName:   cast.ToString(re["columnName"]),
			IndexType:    cast.ToString(re["indexType"]),
			IndexComment: cast.ToString(re["indexComment"]),
			IsUnique:     cast.ToInt(re["isUnique"]) == 1,
			SeqInIndex:   cast.ToInt(re["seqInIndex"]),
			IsPrimaryKey: cast.ToInt(re["isPrimaryKey"]) == 1,
			Extra:        collx.Kvs(IndexSubPartKey, cast.ToInt(re[IndexSubPartKey])),
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
func (md *MysqlMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return dbi.GenTableDDL(md.dc.GetDialect(), md, tableName, dropBeforeCreate)
}

func (md *MysqlMetadata) GetSchemas() ([]string, error) {
	return nil, errors.New("不支持schema")
}
