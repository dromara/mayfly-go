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
	DM_META_FILE      = "metasql/dm_meta.sql"
	DM_DB_SCHEMAS     = "DM_DB_SCHEMAS"
	DM_TABLE_INFO_KEY = "DM_TABLE_INFO"
	DM_INDEX_INFO_KEY = "DM_INDEX_INFO"
	DM_COLUMN_MA_KEY  = "DM_COLUMN_MA"
)

type DMMetaData struct {
	dbi.DefaultMetaData

	dc *dbi.DbConn
}

func (dd *DMMetaData) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := dd.dc.Query("select * from v$instance")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["SVR_VERSION"]),
	}
	return ds, nil
}

func (dd *DMMetaData) GetDbNames() ([]string, error) {
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

func (dd *DMMetaData) GetTables(tableNames ...string) ([]dbi.Table, error) {
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.RemoveQuote(dd, val))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(DM_META_FILE, DM_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = dd.dc.Query(sql)
	if err != nil {
		return nil, err
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
func (dd *DMMetaData) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.RemoveQuote(dd, val))
	}), ",")

	_, res, err := dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columnHelper := dd.dc.GetMetaData().GetColumnHelper()
	columns := make([]dbi.Column, 0)
	for _, re := range res {
		column := dbi.Column{
			TableName:     cast.ToString(re["TABLE_NAME"]),
			ColumnName:    cast.ToString(re["COLUMN_NAME"]),
			DataType:      dbi.ColumnDataType(anyx.ToString(re["DATA_TYPE"])),
			CharMaxLength: cast.ToInt(re["CHAR_MAX_LENGTH"]),
			ColumnComment: cast.ToString(re["COLUMN_COMMENT"]),
			Nullable:      cast.ToString(re["NULLABLE"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["IS_PRIMARY_KEY"]) == 1,
			IsIdentity:    cast.ToInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: cast.ToString(re["COLUMN_DEFAULT"]),
			NumPrecision:  cast.ToInt(re["NUM_PRECISION"]),
			NumScale:      cast.ToInt(re["NUM_SCALE"]),
		}
		columnHelper.FixColumn(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

func (dd *DMMetaData) GetPrimaryKey(tablename string) (string, error) {
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
func (dd *DMMetaData) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
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

func (dd *DMMetaData) genColumnBasicSql(column dbi.Column) string {
	meta := dd.dc.GetMetaData()
	colName := meta.QuoteIdentifier(column.ColumnName)
	dataType := string(column.DataType)

	incr := ""
	if column.IsIdentity {
		incr = " IDENTITY"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(dataType)) {
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

func (dd *DMMetaData) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
	meta := dd.dc.GetMetaData()
	sqls := make([]string, 0)
	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}

		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, name := range cols {
			colNames[i] = meta.QuoteIdentifier(name)
		}

		sqls = append(sqls, fmt.Sprintf("create %s index %s on %s(%s)", unique, index.IndexName, meta.QuoteIdentifier(tableInfo.TableName), strings.Join(colNames, ",")))
	}
	return sqls
}

func (dd *DMMetaData) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	meta := dd.dc.GetMetaData()
	tbName := meta.QuoteIdentifier(tableInfo.TableName)
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("drop table if exists %s", tbName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("create table %s (", tbName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, meta.QuoteIdentifier(column.ColumnName))
		}
		fields = append(fields, dd.genColumnBasicSql(column))
		if column.ColumnComment != "" {
			comment := meta.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf("comment on column %s.%s is '%s'", tbName, meta.QuoteIdentifier(column.ColumnName), comment))
		}
	}
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(",\n PRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	tableCommentSql := ""
	if tableInfo.TableComment != "" {
		comment := meta.QuoteEscape(tableInfo.TableComment)
		tableCommentSql = fmt.Sprintf("comment on table %s is '%s'", tbName, comment)
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

// 获取建表ddl
func (dd *DMMetaData) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {

	// 1.获取表信息
	tbs, err := dd.GetTables(tableName)
	tableInfo := &dbi.Table{}
	if err != nil || tbs == nil || len(tbs) <= 0 {
		logx.Errorf("获取表信息失败, %s", tableName)
		return "", err
	}
	tableInfo.TableName = tbs[0].TableName
	tableInfo.TableComment = tbs[0].TableComment

	// 2.获取列信息
	columns, err := dd.GetColumns(tableName)
	if err != nil {
		logx.Errorf("获取列信息失败, %s", tableName)
		return "", err
	}
	tableDDLArr := dd.GenerateTableDDL(columns, *tableInfo, dropBeforeCreate)
	// 3.获取索引信息
	indexs, err := dd.GetTableIndex(tableName)
	if err != nil {
		logx.Errorf("获取索引信息失败, %s", tableName)
		return "", err
	}
	// 组装返回
	tableDDLArr = append(tableDDLArr, dd.GenerateIndexDDL(indexs, *tableInfo)...)
	return strings.Join(tableDDLArr, ";\n"), nil
}

// 获取DM当前连接的库可访问的schemaNames
func (dd *DMMetaData) GetSchemas() ([]string, error) {
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

func (dd *DMMetaData) GetDataHelper() dbi.DataHelper {
	return new(DataHelper)
}

func (dd *DMMetaData) GetColumnHelper() dbi.ColumnHelper {
	return new(ColumnHelper)
}

func (dd *DMMetaData) GetDumpHelper() dbi.DumpHelper {
	return new(DumpHelper)
}
