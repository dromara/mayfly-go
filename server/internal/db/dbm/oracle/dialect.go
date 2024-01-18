package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
	"reflect"
	"regexp"
	"strings"
	"time"

	_ "gitee.com/chunanyong/dm"
)

// ---------------------------------- DM元数据 -----------------------------------
const (
	ORACLE_META_FILE      = "metasql/oracle_meta.sql"
	ORACLE_DB_SCHEMAS     = "ORACLE_DB_SCHEMAS"
	ORACLE_TABLE_INFO_KEY = "ORACLE_TABLE_INFO"
	ORACLE_INDEX_INFO_KEY = "ORACLE_INDEX_INFO"
	ORACLE_COLUMN_MA_KEY  = "ORACLE_COLUMN_MA"
)

type OracleDialect struct {
	dc *dbi.DbConn
}

func (od *OracleDialect) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := od.dc.Query("select * from v$instance")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["VERSION"]),
	}
	return ds, nil
}

func (od *OracleDialect) GetDbNames() ([]string, error) {
	_, res, err := od.dc.Query("SELECT name AS DBNAME FROM v$database")
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
func (od *OracleDialect) GetTables() ([]dbi.Table, error) {

	// 首先执行更新统计信息sql 这个统计信息在数据量比较大的时候就比较耗时，所以最好定时执行
	// _, _, err := pd.dc.Query("dbms_stats.GATHER_SCHEMA_stats(SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID))")

	// 查询表信息
	_, res, err := od.dc.Query(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_TABLE_INFO_KEY))
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
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
func (od *OracleDialect) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	if len(tableNames) == 0 {
		// 处理空切片的情况，返回错误
		return nil, errorx.NewBiz("获取表名失败")
	}

	rawTableNames := make([]string, 0, len(tableNames))

	for _, name := range tableNames {
		// 如果表名已转译，则删除头尾的反引号
		if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
			name = name[1 : len(name)-1]
		}
		rawTableNames = append(rawTableNames, fmt.Sprintf("'%s'", name))
	}

	tableName := strings.Join(rawTableNames, ", ")

	_, res, err := od.dc.Query(fmt.Sprintf(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		defaultVal := anyx.ConvString(re["COLUMN_DEFAULT"])
		// 如果默认值包含.nextval，说明是序列，默认值为null
		if strings.Contains(defaultVal, ".nextval") {
			defaultVal = ""
		}
		columns = append(columns, dbi.Column{
			TableName:     re["TABLE_NAME"].(string),
			ColumnName:    re["COLUMN_NAME"].(string),
			ColumnType:    anyx.ConvString(re["COLUMN_TYPE"]),
			ColumnComment: anyx.ConvString(re["COLUMN_COMMENT"]),
			Nullable:      anyx.ConvString(re["NULLABLE"]),
			ColumnKey:     anyx.ConvString(re["COLUMN_KEY"]),
			ColumnDefault: defaultVal,
			NumScale:      anyx.ConvString(re["NUM_SCALE"]),
		})
	}
	return columns, nil
}

func (od *OracleDialect) GetPrimaryKey(tablename string) (string, error) {
	columns, err := od.GetColumns(tablename)
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
func (od *OracleDialect) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := od.dc.Query(fmt.Sprintf(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    re["INDEX_NAME"].(string),
			ColumnName:   anyx.ConvString(re["COLUMN_NAME"]),
			IndexType:    anyx.ConvString(re["INDEX_TYPE"]),
			IndexComment: anyx.ConvString(re["INDEX_COMMENT"]),
			NonUnique:    anyx.ConvInt(re["NON_UNIQUE"]),
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
func (od *OracleDialect) GetTableDDL(tableName string) (string, error) {
	ddlSql := fmt.Sprintf("SELECT DBMS_METADATA.GET_DDL('TABLE', '%s', (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual)) AS TABLE_DDL FROM DUAL", tableName)
	_, res, err := od.dc.Query(ddlSql)
	if err != nil {
		return "", err
	}
	// 建表ddl
	var builder strings.Builder
	for _, re := range res {
		builder.WriteString(re["TABLE_DDL"].(string))
	}

	// 表注释
	_, res, err = od.dc.Query(fmt.Sprintf(`
			select OWNER, COMMENTS from ALL_TAB_COMMENTS where TABLE_TYPE='TABLE' and TABLE_NAME = '%s'
		    and owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual) `, tableName))
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
		FROM ALL_COL_COMMENTS
		WHERE OWNER = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual)
		  AND TABLE_NAME = '%s'
		`, tableName)
	_, res, err = od.dc.Query(fieldSql)
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
		select DBMS_METADATA.GET_DDL('INDEX', a.INDEX_NAME, a.OWNER) AS INDEX_DEF from ALL_INDEXES a
		join ALL_objects b on a.owner = b.owner and b.object_name = a.index_name and b.object_type = 'INDEX'
		where a.owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual)
		and a.table_name = '%s' 
	`, tableName)
	_, res, err = od.dc.Query(indexSql)
	if err != nil {
		return "", err
	}
	for _, re := range res {
		builder.WriteString("\n\n" + re["INDEX_DEF"].(string))
	}

	return builder.String(), nil
}

func (od *OracleDialect) WalkTableRecord(tableName string, walkFn dbi.WalkQueryRowsFunc) error {
	return od.dc.WalkQueryRows(context.Background(), fmt.Sprintf("SELECT * FROM %s", tableName), walkFn)
}

// 获取DM当前连接的库可访问的schemaNames
func (od *OracleDialect) GetSchemas() ([]string, error) {
	sql := dbi.GetLocalSql(ORACLE_META_FILE, ORACLE_DB_SCHEMAS)
	_, res, err := od.dc.Query(sql)
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
func (od *OracleDialect) GetDbProgram() dbi.DbProgram {
	panic("implement me")
}

func (od *OracleDialect) GetDataType(dbColumnType string) dbi.DataType {
	if regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`).MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	// 日期时间类型
	if regexp.MustCompile(`(?i)datetime|timestamp`).MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	// 日期类型
	if regexp.MustCompile(`(?i)date`).MatchString(dbColumnType) {
		return dbi.DataTypeDate
	}
	// 时间类型
	if regexp.MustCompile(`(?i)time`).MatchString(dbColumnType) {
		return dbi.DataTypeTime
	}
	return dbi.DataTypeString
}

func (od *OracleDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	//INSERT ALL
	//INTO my_table(field_1,field_2) VALUES (value_1,value_2)
	//INTO my_table(field_1,field_2) VALUES (value_3,value_4)
	//INTO my_table(field_1,field_2) VALUES (value_5,value_6)
	//SELECT 1 FROM DUAL;

	if len(values) <= 0 {
		return 0, nil
	}

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}

	// 拼接oracle批量插入语句
	sqlArr := make([]string, 0)
	sqlArr = append(sqlArr, "INSERT ALL")

	// 拼接带占位符的sql oracle的占位符是:1,:2,:3....
	for i := 0; i < len(args); i += len(columns) {
		var placeholder []string
		for j := 0; j < len(columns); j++ {
			// 判断字符串数据格式是时间"2023-06-25 10:40:10"  占位符需要变成 to_date(:x, 'fmt')
			if reflect.TypeOf(args[i+j]) == reflect.TypeOf("") {
				if regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`).MatchString(args[i+j].(string)) {
					placeholder = append(placeholder, fmt.Sprintf("to_date(:%d, 'yyyy-mm-dd hh24:mi:ss')", i+j+1))
				} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(args[i+j].(string)) {
					// 只有年月日的数据，oracle会自动补零时分秒，如：2024-01-02: to_date('2024-01-02','yyyy-mm-dd') 输出：2024-01-02 00:00:00
					placeholder = append(placeholder, fmt.Sprintf("to_date(:%d, 'yyyy-mm-dd')", i+j+1))
				} else if regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`).MatchString(args[i+j].(string)) {
					// 只有时间的数据，oracle会拼接当前月份的年月日，如当前月份是2024-01: to_date('13:23:11','hh24:mi:ss') 输出：2024-01-01 13:23:11
					placeholder = append(placeholder, fmt.Sprintf("to_date(:%d, 'hh24:mi:ss')", i+j+1))
				}
				continue
			}

			placeholder = append(placeholder, fmt.Sprintf(":%d", i+j+1))
		}
		sqlArr = append(sqlArr, fmt.Sprintf("INTO %s (%s) VALUES (%s)", od.dc.Info.Type.QuoteIdentifier(tableName), strings.Join(columns, ","), strings.Join(placeholder, ",")))
	}
	sqlArr = append(sqlArr, "SELECT 1 FROM DUAL")

	// 执行批量insert sql
	res, err := od.dc.TxExec(tx, strings.Join(sqlArr, " "), args...)
	return res, err
}

func (od *OracleDialect) FormatStrData(dbColumnValue string, dataType dbi.DataType) string {
	switch dataType {
	case dbi.DataTypeDateTime: // "2024-01-02T22:08:22.275697+08:00"
		res, _ := time.Parse(time.RFC3339, dbColumnValue)
		return res.Format(time.DateTime)
	case dbi.DataTypeDate: // "2024-01-02T00:00:00+08:00"
		res, _ := time.Parse(time.RFC3339, dbColumnValue)
		return res.Format(time.DateOnly)
	case dbi.DataTypeTime: // "0000-01-01T22:08:22.275688+08:00"
		res, _ := time.Parse(time.RFC3339, dbColumnValue)
		return res.Format(time.TimeOnly)
	}
	return dbColumnValue
}
