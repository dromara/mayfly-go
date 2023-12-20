package dbm

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
	"strings"

	_ "gitee.com/chunanyong/dm"
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

func (dd *DMDialect) GetDbServer() (*DbServer, error) {
	_, res, err := dd.dc.Query("select * from v$instance")
	if err != nil {
		return nil, err
	}
	ds := &DbServer{
		Version: anyx.ConvString(res[0]["SVR_VERSION"]),
	}
	return ds, nil
}

func (pd *DMDialect) GetDbNames() ([]string, error) {
	_, res, err := pd.dc.Query("SELECT name AS DBNAME FROM v$database")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, anyx.ConvString(re["DBNAME"]))
	}

	return databases, nil
}

// 获取表基础元信息, 如表名等
func (pd *DMDialect) GetTables() ([]Table, error) {

	// 首先执行更新统计信息sql 这个统计信息在数据量比较大的时候就比较耗时，所以最好定时执行
	// _, _, err := pd.dc.Query("dbms_stats.GATHER_SCHEMA_stats(SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))")

	// 查询表信息
	_, res, err := pd.dc.Query(GetLocalSql(DM_META_FILE, DM_TABLE_INFO_KEY))
	if err != nil {
		return nil, err
	}

	tables := make([]Table, 0)
	for _, re := range res {
		tables = append(tables, Table{
			TableName:    re["TABLE_NAME"].(string),
			TableComment: anyx.ConvString(re["TABLE_COMMENT"]),
			CreateTime:   anyx.ConvString(re["CREATE_TIME"]),
			TableRows:    anyx.ConvInt(re["TABLE_ROWS"]),
			DataLength:   anyx.ConvInt64(re["DATA_LENGTH"]),
			IndexLength:  anyx.ConvInt64(re["INDEX_LENGTH"]),
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
			TableName:     re["TABLE_NAME"].(string),
			ColumnName:    re["COLUMN_NAME"].(string),
			ColumnType:    anyx.ConvString(re["COLUMN_TYPE"]),
			ColumnComment: anyx.ConvString(re["COLUMN_COMMENT"]),
			Nullable:      anyx.ConvString(re["NULLABLE"]),
			ColumnKey:     anyx.ConvString(re["COLUMN_KEY"]),
			ColumnDefault: anyx.ConvString(re["COLUMN_DEFAULT"]),
			NumScale:      anyx.ConvString(re["NUM_SCALE"]),
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
			IndexName:    re["INDEX_NAME"].(string),
			ColumnName:   anyx.ConvString(re["COLUMN_NAME"]),
			IndexType:    anyx.ConvString(re["INDEX_TYPE"]),
			IndexComment: anyx.ConvString(re["INDEX_COMMENT"]),
			NonUnique:    anyx.ConvInt(re["NON_UNIQUE"]),
			SeqInIndex:   anyx.ConvInt(re["SEQ_IN_INDEX"]),
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
func (pd *DMDialect) GetTableDDL(tableName string) (string, error) {
	ddlSql := fmt.Sprintf("CALL SP_TABLEDEF((SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)), '%s')", tableName)
	_, res, err := pd.dc.Query(ddlSql)
	if err != nil {
		return "", err
	}
	// 建表ddl
	var builder strings.Builder
	for _, re := range res {
		builder.WriteString(re["COLUMN_VALUE"].(string))
	}

	// 表注释
	_, res, err = pd.dc.Query(fmt.Sprintf(`
			select OWNER, COMMENTS from DBA_TAB_COMMENTS where TABLE_TYPE='TABLE' and TABLE_NAME = '%s'
		    and owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
			                                      `, tableName))
	if err != nil {
		return "", err
	}
	for _, re := range res {
		// COMMENT ON TABLE "SYS_MENU" IS '菜单表';
		if re["COMMENTS"] != nil {
			tableComment := fmt.Sprintf("\n\nCOMMENT ON TABLE \"%s\".\"%s\" IS '%s';", re["OWNER"].(string), tableName, re["COMMENTS"].(string))
			builder.WriteString(tableComment)
		}
	}

	// 字段注释
	fieldSql := fmt.Sprintf(`
		SELECT OWNER, COLUMN_NAME, COMMENTS
		FROM USER_COL_COMMENTS
		WHERE OWNER = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
		  AND TABLE_NAME = '%s'
		`, tableName)
	_, res, err = pd.dc.Query(fieldSql)
	if err != nil {
		return "", err
	}

	builder.WriteString("\n")
	for _, re := range res {
		// COMMENT ON COLUMN "SYS_MENU"."BIZ_CODE" IS '业务编码，应用编码1';
		if re["COMMENTS"] != nil {
			fieldComment := fmt.Sprintf("\nCOMMENT ON COLUMN \"%s\".\"%s\".\"%s\" IS '%s';", re["OWNER"].(string), tableName, re["COLUMN_NAME"].(string), re["COMMENTS"].(string))
			builder.WriteString(fieldComment)
		}
	}

	// 索引信息
	indexSql := fmt.Sprintf(`
		select indexdef(b.object_id,1) as INDEX_DEF from DBA_INDEXES a
		join dba_objects b on a.owner = b.owner and b.object_name = a.index_name and b.object_type = 'INDEX'
		where a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
		and a.table_name = '%s' 
		and indexdef(b.object_id,1) != '禁止查看系统定义的索引信息'
	`, tableName)
	_, res, err = pd.dc.Query(indexSql)
	if err != nil {
		return "", err
	}
	for _, re := range res {
		builder.WriteString("\n\n" + re["INDEX_DEF"].(string))
	}

	return builder.String(), nil
}

func (pd *DMDialect) GetTableRecord(tableName string, pageNum, pageSize int) ([]*QueryColumn, []map[string]any, error) {
	return pd.dc.Query(fmt.Sprintf("SELECT * FROM %s OFFSET %d LIMIT %d", tableName, (pageNum-1)*pageSize, pageSize))
}

func (pd *DMDialect) WalkTableRecord(tableName string, walk func(record map[string]any, columns []*QueryColumn)) error {
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
		schemaNames = append(schemaNames, anyx.ConvString(re["SCHEMA_NAME"]))
	}
	return schemaNames, nil
}
