package mysql

import (
	"errors"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/kanzihuang/vitess/go/vt/sqlparser"
	"github.com/may-fly/cast"
)

const (
	MYSQL_META_FILE      = "metasql/mysql_meta.sql"
	MYSQL_DBS            = "MYSQL_DBS"
	MYSQL_TABLE_INFO_KEY = "MYSQL_TABLE_INFO"
	MYSQL_INDEX_INFO_KEY = "MYSQL_INDEX_INFO"
	MYSQL_COLUMN_MA_KEY  = "MYSQL_COLUMN_MA"
)

type MysqlMetaData struct {
	dbi.DefaultMetaData

	dc *dbi.DbConn
}

func (md *MysqlMetaData) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := md.dc.Query("SELECT VERSION() version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["version"]),
	}
	return ds, nil
}

func (md *MysqlMetaData) GetDbNames() ([]string, error) {
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

func (md *MysqlMetaData) GetTables(tableNames ...string) ([]dbi.Table, error) {
	meta := md.dc.GetMetaData()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", meta.RemoveQuote(val))
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
func (md *MysqlMetaData) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	meta := md.dc.GetMetaData()
	columnHelper := meta.GetColumnHelper()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", meta.RemoveQuote(val))
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
			DataType:      dbi.ColumnDataType(cast.ToString(re["dataType"])),
			ColumnComment: cast.ToString(re["columnComment"]),
			Nullable:      cast.ToString(re["nullable"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["isPrimaryKey"]) == 1,
			IsIdentity:    cast.ToInt(re["isIdentity"]) == 1,
			ColumnDefault: cast.ToString(re["columnDefault"]),
			CharMaxLength: cast.ToInt(re["charMaxLength"]),
			NumPrecision:  cast.ToInt(re["numPrecision"]),
			NumScale:      cast.ToInt(re["numScale"]),
		}

		columnHelper.FixColumn(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

// 获取表主键字段名，不存在主键标识则默认第一个字段
func (md *MysqlMetaData) GetPrimaryKey(tablename string) (string, error) {
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
func (md *MysqlMetaData) GetTableIndex(tableName string) ([]dbi.Index, error) {
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

// 获取建索引ddl
func (md *MysqlMetaData) GenerateIndexDDL(indexs []dbi.Index, tableInfo dbi.Table) []string {
	meta := md.dc.GetMetaData()
	sqlArr := make([]string, 0)
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
		sqlTmp := "ALTER TABLE %s ADD %s INDEX %s(%s) USING BTREE"
		sqlStr := fmt.Sprintf(sqlTmp, meta.QuoteIdentifier(tableInfo.TableName), unique, meta.QuoteIdentifier(index.IndexName), strings.Join(colNames, ","))
		comment := meta.QuoteEscape(index.IndexComment)
		if comment != "" {
			sqlStr += fmt.Sprintf(" COMMENT '%s'", comment)
		}
		sqlArr = append(sqlArr, sqlStr)
	}
	return sqlArr
}

func (md *MysqlMetaData) genColumnBasicSql(column dbi.Column) string {
	meta := md.dc.GetMetaData()
	dataType := string(column.DataType)

	incr := ""
	if column.IsIdentity {
		incr = " AUTO_INCREMENT"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}
	columnType := column.GetColumnType()
	if nullAble == "" && strings.Contains(columnType, "timestamp") {
		nullAble = " NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的
	if column.ColumnDefault != "" &&
		// 当默认值是字符串'NULL'时，不需要设置默认值
		column.ColumnDefault != "NULL" &&
		// 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
		!strings.Contains(column.ColumnDefault, "(") {
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
	comment := ""
	if column.ColumnComment != "" {
		// 防止注释内含有特殊字符串导致sql出错
		commentStr := meta.QuoteEscape(column.ColumnComment)
		comment = fmt.Sprintf(" COMMENT '%s'", commentStr)
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s%s", md.dc.GetMetaData().QuoteIdentifier(column.ColumnName), columnType, nullAble, incr, defVal, comment)
	return columnSql
}

// 获取建表ddl
func (md *MysqlMetaData) GenerateTableDDL(columns []dbi.Column, tableInfo dbi.Table, dropBeforeCreate bool) []string {
	meta := md.dc.GetMetaData()
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", meta.QuoteIdentifier(tableInfo.TableName)))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", meta.QuoteIdentifier(tableInfo.TableName))
	fields := make([]string, 0)
	pks := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, column.ColumnName)
		}
		fields = append(fields, md.genColumnBasicSql(column))
	}

	// 建表ddl
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	// 表注释
	if tableInfo.TableComment != "" {
		createSql += fmt.Sprintf(" COMMENT '%s'", meta.QuoteEscape(tableInfo.TableComment))
	}

	sqlArr = append(sqlArr, createSql)

	return sqlArr
}

// 获取建表ddl
func (md *MysqlMetaData) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	// 1.获取表信息
	tbs, err := md.GetTables(tableName)
	tableInfo := &dbi.Table{}
	if err != nil || tbs == nil || len(tbs) <= 0 {
		logx.Errorf("获取表信息失败, %s", tableName)
		return "", err
	}
	tableInfo.TableName = tbs[0].TableName
	tableInfo.TableComment = tbs[0].TableComment

	// 2.获取列信息
	columns, err := md.GetColumns(tableName)
	if err != nil {
		logx.Errorf("获取列信息失败, %s", tableName)
		return "", err
	}
	tableDDLArr := md.GenerateTableDDL(columns, *tableInfo, dropBeforeCreate)
	// 3.获取索引信息
	indexs, err := md.GetTableIndex(tableName)
	if err != nil {
		logx.Errorf("获取索引信息失败, %s", tableName)
		return "", err
	}
	// 组装返回
	tableDDLArr = append(tableDDLArr, md.GenerateIndexDDL(indexs, *tableInfo)...)
	return strings.Join(tableDDLArr, ";\n"), nil
}

func (md *MysqlMetaData) GetSchemas() ([]string, error) {
	return nil, errors.New("不支持schema")
}

func (md *MysqlMetaData) GetIdentifierQuoteString() string {
	return "`"
}

func (md *MysqlMetaData) QuoteLiteral(literal string) string {
	literal = strings.ReplaceAll(literal, `\`, `\\`)
	literal = strings.ReplaceAll(literal, `'`, `''`)
	return "'" + literal + "'"
}

func (md *MysqlMetaData) GetSqlParserDialect() sqlparser.Dialect {
	return sqlparser.MysqlDialect{}
}

func (md *MysqlMetaData) GetDataHelper() dbi.DataHelper {
	return new(DataHelper)
}

func (md *MysqlMetaData) GetColumnHelper() dbi.ColumnHelper {
	return new(ColumnHelper)
}
