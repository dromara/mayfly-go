package dm

import (
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"strings"
	"time"

	"github.com/kanzihuang/vitess/go/vt/sqlparser"

	_ "gitee.com/chunanyong/dm"
)

const (
	DM_META_FILE      = "metasql/dm_meta.sql"
	DM_DB_SCHEMAS     = "DM_DB_SCHEMAS"
	DM_TABLE_INFO_KEY = "DM_TABLE_INFO"
	DM_INDEX_INFO_KEY = "DM_INDEX_INFO"
	DM_COLUMN_MA_KEY  = "DM_COLUMN_MA"
)

type DMDialect struct {
	dc *dbi.DbConn
}

func (dd *DMDialect) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := dd.dc.Query("select * from v$instance")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["SVR_VERSION"]),
	}
	return ds, nil
}

func (dd *DMDialect) GetDbNames() ([]string, error) {
	_, res, err := dd.dc.Query("SELECT name AS DBNAME FROM v$database")
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
func (dd *DMDialect) GetTables() ([]dbi.Table, error) {

	// 首先执行更新统计信息sql 这个统计信息在数据量比较大的时候就比较耗时，所以最好定时执行
	// _, _, err := pd.dc.Query("dbms_stats.GATHER_SCHEMA_stats(SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))")

	// 查询表信息
	_, res, err := dd.dc.Query(dbi.GetLocalSql(DM_META_FILE, DM_TABLE_INFO_KEY))
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    anyx.ConvString(re["TABLE_NAME"]),
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
func (dd *DMDialect) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dbType := dd.dc.Info.Type
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbType.RemoveQuote(val))
	}), ",")

	_, res, err := dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		columns = append(columns, dbi.Column{
			TableName:     anyx.ConvString(re["TABLE_NAME"]),
			ColumnName:    anyx.ConvString(re["COLUMN_NAME"]),
			ColumnType:    anyx.ConvString(re["COLUMN_TYPE"]),
			ColumnComment: anyx.ConvString(re["COLUMN_COMMENT"]),
			Nullable:      anyx.ConvString(re["NULLABLE"]),
			IsPrimaryKey:  anyx.ConvInt(re["IS_PRIMARY_KEY"]) == 1,
			IsIdentity:    anyx.ConvInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: anyx.ConvString(re["COLUMN_DEFAULT"]),
			NumScale:      anyx.ConvString(re["NUM_SCALE"]),
		})
	}
	return columns, nil
}

func (dd *DMDialect) GetPrimaryKey(tablename string) (string, error) {
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
func (dd *DMDialect) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := dd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(DM_META_FILE, DM_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    anyx.ConvString(re["INDEX_NAME"]),
			ColumnName:   anyx.ConvString(re["COLUMN_NAME"]),
			IndexType:    anyx.ConvString(re["INDEX_TYPE"]),
			IndexComment: anyx.ConvString(re["INDEX_COMMENT"]),
			IsUnique:     anyx.ConvInt(re["IS_UNIQUE"]) == 1,
			SeqInIndex:   anyx.ConvInt(re["SEQ_IN_INDEX"]),
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
func (dd *DMDialect) GetTableDDL(tableName string) (string, error) {
	ddlSql := fmt.Sprintf("CALL SP_TABLEDEF((SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID)), '%s')", tableName)
	_, res, err := dd.dc.Query(ddlSql)
	if err != nil {
		return "", err
	}
	// 建表ddl
	var builder strings.Builder
	for _, re := range res {
		builder.WriteString(re["COLUMN_VALUE"].(string))
	}

	// 表注释
	_, res, err = dd.dc.Query(fmt.Sprintf(`
			select OWNER, COMMENTS from ALL_TAB_COMMENTS where TABLE_TYPE='TABLE' and TABLE_NAME = '%s'
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
	_, res, err = dd.dc.Query(fieldSql)
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
		select indexdef(b.object_id,1) as INDEX_DEF from ALL_INDEXES a
		join ALL_objects b on a.owner = b.owner and b.object_name = a.index_name and b.object_type = 'INDEX'
		where a.owner = (SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))
		and a.table_name = '%s' 
		and indexdef(b.object_id,1) != '禁止查看系统定义的索引信息'
	`, tableName)
	_, res, err = dd.dc.Query(indexSql)
	if err != nil {
		return "", err
	}
	for _, re := range res {
		builder.WriteString("\n\n" + re["INDEX_DEF"].(string))
	}

	return builder.String(), nil
}

// 获取DM当前连接的库可访问的schemaNames
func (dd *DMDialect) GetSchemas() ([]string, error) {
	sql := dbi.GetLocalSql(DM_META_FILE, DM_DB_SCHEMAS)
	_, res, err := dd.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, anyx.ConvString(re["SCHEMA_NAME"]))
	}
	return schemaNames, nil
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (dd *DMDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, fmt.Errorf("该数据库类型不支持数据库备份与恢复: %v", dd.dc.Info.Type)
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

func (dd *DMDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	// 执行批量insert sql
	// insert into "table_name" ("column1", "column2", ...) values (value1, value2, ...)

	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	sqlTemp := fmt.Sprintf("insert into %s (%s) values %s", dd.dc.Info.Type.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)
	effRows := 0
	for _, value := range values {
		// 达梦数据库只能一条条的执行insert
		er, err := dd.dc.TxExec(tx, sqlTemp, value...)
		if err != nil {
			logx.Errorf("执行sql失败：%s", err.Error())
			return int64(effRows), err
		}
		effRows += int(er)
	}
	// 执行批量insert sql
	return int64(effRows), nil
}

func (dd *DMDialect) GetDataConverter() dbi.DataConverter {
	return new(DataConverter)
}

type DataConverter struct {
}

func (dd *DataConverter) GetDataType(dbColumnType string) dbi.DataType {
	if numberRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	if datetimeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	if dateRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDate
	}
	if timeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeTime
	}
	return dbi.DataTypeString
}

func (dd *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
	str := anyx.ToString(dbColumnValue)
	switch dataType {
	case dbi.DataTypeDateTime: // "2024-01-02T22:08:22.275697+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.DateTime)
	case dbi.DataTypeDate: // "2024-01-02T00:00:00+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.DateOnly)
	case dbi.DataTypeTime: // "0000-01-01T22:08:22.275688+08:00"
		res, _ := time.Parse(time.RFC3339, str)
		return res.Format(time.TimeOnly)
	}
	return str
}

func (dd *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	return dbColumnValue
}

func (dd *DMDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName
	ddl, err := dd.GetTableDDL(tableName)
	if err != nil {
		return err
	}
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// 替换新表名
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("\"%s\"", strings.ToUpper(tableName)), fmt.Sprintf("\"%s\"", strings.ToUpper(newTableName)))
	// 去除空格换行
	ddl = stringx.TrimSpaceAndBr(ddl)
	sqls, err := sqlparser.SplitStatementToPieces(ddl, sqlparser.WithDialect(dd.dc.Info.Type.Dialect()))
	for _, sql := range sqls {
		_, _ = dd.dc.Exec(sql)
	}

	// 复制数据
	if copy.CopyData {
		go func() {
			// 设置允许填充自增列之后，显示指定列名可以插入自增列
			_, _ = dd.dc.Exec(fmt.Sprintf("set identity_insert \"%s\" on", newTableName))
			// 获取列名
			columns, _ := dd.GetColumns(tableName)
			columnArr := make([]string, 0)
			for _, column := range columns {
				columnArr = append(columnArr, fmt.Sprintf("\"%s\"", column.ColumnName))
			}
			columnStr := strings.Join(columnArr, ",")
			// 插入新数据并显示指定列
			_, _ = dd.dc.Exec(fmt.Sprintf("insert into \"%s\" (%s) select %s from \"%s\"", newTableName, columnStr, columnStr, tableName))

			// 执行完成后关闭允许填充自增列
			_, _ = dd.dc.Exec(fmt.Sprintf("set identity_insert \"%s\" off", newTableName))
		}()
	}

	return err
}
