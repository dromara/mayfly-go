package dbm

import (
	"context"
	"database/sql"
	"fmt"
	_ "gitee.com/chunanyong/dm"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
	"strings"
)

func getDmDB(d *DbInfo) (*sql.DB, error) {
	driverName := "dm"
	// SSH Conect 暂时不支持隧道连接
	db := d.Database
	var dbParam string
	if db != "" {
		// postgres database可以使用db/schema表示，方便连接指定schema, 若不存在schema则使用默认schema
		ss := strings.Split(db, "/")
		if len(ss) > 1 {
			dbParam = fmt.Sprintf("%s?schema=%s", ss[0], ss[len(ss)-1])
		} else {
			dbParam = db
		}
	}
	dsn := fmt.Sprintf("dm://%s:%s@%s:%d/%s", d.Username, d.Password, d.Host, d.Port, dbParam)
	return sql.Open(driverName, dsn)
}

// ---------------------------------- DM元数据 -----------------------------------
const (
	DM_META_FILE      = "metasql/dm_meta.sql"
	DM_DB_SCHEMAS     = "DM_DB_SCHEMAS"
	DM_TABLE_INFO_KEY = "DM_TABLE_INFO"
	DM_INDEX_INFO_KEY = "DM_INDEX_INFO"
	DM_COLUMN_MA_KEY  = "DM_COLUMN_MA"
)

type DMDialect struct {
	dc *DbConn
}

func (pd *DMDialect) GetDbNames() ([]string, error) {
	_, res, err := pd.dc.Query("SELECT name AS dbname FROM v$database")
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
func (pd *DMDialect) GetTables() ([]Table, error) {
	_, res, err := pd.dc.Query(GetLocalSql(DM_META_FILE, DM_TABLE_INFO_KEY))
	if err != nil {
		return nil, err
	}

	tables := make([]Table, 0)
	for _, re := range res {
		tables = append(tables, Table{
			TableName:    re["tableName"].(string),
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
func (pd *DMDialect) GetColumns(tableNames ...string) ([]Column, error) {
	tableName := ""
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}

	_, res, err := pd.dc.Query(fmt.Sprintf(GetLocalSql(DM_META_FILE, DM_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]Column, 0)
	for _, re := range res {
		columns = append(columns, Column{
			TableName:     re["tableName"].(string),
			ColumnName:    re["columnName"].(string),
			ColumnType:    anyx.ConvString(re["columnType"]),
			ColumnComment: anyx.ConvString(re["columnComment"]),
			Nullable:      anyx.ConvString(re["nullable"]),
			ColumnKey:     anyx.ConvString(re["columnKey"]),
			ColumnDefault: anyx.ConvString(re["columnDefault"]),
			NumScale:      anyx.ConvString(re["numScale"]),
		})
	}
	return columns, nil
}

func (pd *DMDialect) GetPrimaryKey(tablename string) (string, error) {
	columns, err := pd.GetColumns(tablename)
	if err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", errorx.NewBiz("[%s] 表不存在", tablename)
	}
	for _, v := range columns {
		if v.ColumnKey == "PRI" {
			return v.ColumnName, nil
		}
	}

	return columns[0].ColumnName, nil
}

// 获取表索引信息
func (pd *DMDialect) GetTableIndex(tableName string) ([]Index, error) {
	_, res, err := pd.dc.Query(fmt.Sprintf(GetLocalSql(DM_META_FILE, DM_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]Index, 0)
	for _, re := range res {
		indexs = append(indexs, Index{
			IndexName:    re["indexName"].(string),
			ColumnName:   anyx.ConvString(re["columnName"]),
			IndexType:    anyx.ConvString(re["IndexType"]),
			IndexComment: anyx.ConvString(re["indexComment"]),
			NonUnique:    anyx.ConvInt(re["nonUnique"]),
			SeqInIndex:   anyx.ConvInt(re["seqInIndex"]),
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	result := make([]Index, 0)
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
func (pd *DMDialect) GetCreateTableDdl(tableName string) (string, error) {
	ddlSql := fmt.Sprintf("CALL SP_TABLEDEF((SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)), '%s')", tableName)
	_, res, err := pd.dc.Query(ddlSql)
	if err != nil {
		return "", err
	}

	return res[0]["COLUMN_VALUE"].(string), nil
}

func (pd *DMDialect) GetTableRecord(tableName string, pageNum, pageSize int) ([]string, []map[string]any, error) {
	return pd.dc.Query(fmt.Sprintf("SELECT * FROM %s OFFSET %d LIMIT %d", tableName, (pageNum-1)*pageSize, pageSize))
}

func (pd *DMDialect) WalkTableRecord(tableName string, walk func(record map[string]any, columns []string)) error {
	return pd.dc.WalkTableRecord(context.Background(), fmt.Sprintf("SELECT * FROM %s", tableName), walk)
}

// 获取DM当前连接的库可访问的schemaNames
func (pd *DMDialect) GetSchemas() ([]string, error) {
	sql := GetLocalSql(DM_META_FILE, DM_DB_SCHEMAS)
	_, res, err := pd.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, anyx.ConvString(re["schemaName"]))
	}
	return schemaNames, nil
}
