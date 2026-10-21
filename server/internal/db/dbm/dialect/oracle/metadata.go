package oracle

import (
	_ "embed"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/spf13/cast"
)

//go:embed meta.sql
var metaSQLFile string

// metaSQL 方言元数据SQL模板（按备注key解析并缓存，格式见dbi.SQLTemplates）
var metaSQL = dbi.NewSQLTemplates(metaSQLFile)

// ---------------------------------- DM元数据 -----------------------------------
const (
	ORACLE_DB_SCHEMAS            = "ORACLE_DB_SCHEMAS"
	ORACLE_DB_SCHEMAS_MAINTAINED = "ORACLE_DB_SCHEMAS_MAINTAINED"
	ORACLE_TABLE_INFO_KEY        = "ORACLE_TABLE_INFO"
	ORACLE_INDEX_INFO_KEY        = "ORACLE_INDEX_INFO"
	ORACLE_COLUMN_MA_KEY         = "ORACLE_COLUMN_MA"
)

var (
	_ dbi.ServerInfo       = (*OracleMetadata)(nil)
	_ dbi.MetadataProvider = (*OracleMetadata)(nil)
)

type OracleMetadata struct {
	dbi.DefaultServerInfo

	di *dbi.DbInfo

	version dbi.DbVersion
}

func (od *OracleMetadata11) GetCompatibleDbVersion() dbi.DbVersion {
	return od.version
}

func (od *OracleMetadata) GetDbServer() (*dbi.DbServer, error) {
	// 使用Oracle内置函数获取版本号，任何用户都有权限
	_, res, err := od.di.Query("SELECT dbms_utility.db_version AS VERSION FROM SYS.dual")
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, errorx.NewBiz("failed to get database version: empty result")
	}

	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["VERSION"]),
	}
	dbi.ParseDbVersion(ds)
	return ds, nil
}

func (od *OracleMetadata) GetDbNames() ([]string, error) {
	_, res, err := od.di.Query("SELECT sys_context('USERENV', 'DB_NAME') AS DBNAME FROM SYS.dual")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, cast.ToString(re["DBNAME"]))
	}

	return databases, nil
}

func (od *OracleMetadata) GetTables(tableNames ...string) ([]dbi.Table, error) {
	dialect := od.di.GetDialect()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.QuoteEscape(dialect.Quoter().Trim(val)))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(metaSQL.Get(ORACLE_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = od.di.Query(sql)
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
func (od *OracleMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dialect := od.di.GetDialect()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.QuoteEscape(dialect.Quoter().Trim(val)))
	}), ",")

	// 如果表数量超过了1000，需要分批查询
	if len(tableNames) > 1000 {
		columns := make([]dbi.Column, 0)
		for i := 0; i < len(tableNames); i += 1000 {
			end := i + 1000
			if end > len(tableNames) {
				end = len(tableNames)
			}
			tables := tableNames[i:end]
			cols, err := od.GetColumns(tables...)
			if err != nil {
				return nil, err
			}
			columns = append(columns, cols...)
		}
		return columns, nil
	}

	_, res, err := od.di.Query(fmt.Sprintf(metaSQL.Get(ORACLE_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		column := dbi.Column{
			TableName:     cast.ToString(re["TABLE_NAME"]),
			ColumnName:    cast.ToString(re["COLUMN_NAME"]),
			DataType:      cast.ToString(re["DATA_TYPE"]),
			CharMaxLength: cast.ToInt(re["CHAR_MAX_LENGTH"]),
			ColumnComment: cast.ToString(re["COLUMN_COMMENT"]),
			Nullable:      cast.ToString(re["NULLABLE"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["IS_PRIMARY_KEY"]) == 1,
			AutoIncrement: cast.ToInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: cast.ToString(re["COLUMN_DEFAULT"]),
			NumPrecision:  cast.ToInt(re["NUM_PRECISION"]),
			NumScale:      cast.ToInt(re["NUM_SCALE"]),
		}

		od.di.GetDbDataType(column.DataType).FixColumn(&column)
		// Oracle的DATA_DEFAULT对字面量默认值恒带引号（'abc'），不带引号的函数/运算形态（to_char(...)、sysdate+1）
		// 即表达式默认值，标记后不写入目标DDL，避免被当成字符串默认值静默污染
		dbi.MarkExprDefault(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

func (od *OracleMetadata) GetPrimaryKeys(tablename string) ([]string, error) {
	columns, err := od.GetColumns(tablename)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, errorx.NewBizf("[%s] 表不存在", tablename)
	}
	pks := make([]string, 0, 2)
	for _, v := range columns {
		if v.IsPrimaryKey {
			pks = append(pks, v.ColumnName)
		}
	}

	// 联合主键返回多列；无主键返回空切片（不兜底首列）
	return pks, nil
}

// 获取表索引信息
func (od *OracleMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := od.di.Query(fmt.Sprintf(metaSQL.Get(ORACLE_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexes := make([]dbi.Index, 0)
	for _, re := range res {
		indexes = append(indexes, dbi.Index{
			IndexName:    cast.ToString(re["INDEX_NAME"]),
			ColumnName:   cast.ToString(re["COLUMN_NAME"]),
			IndexType:    cast.ToString(re["INDEX_TYPE"]),
			IndexComment: cast.ToString(re["INDEX_COMMENT"]),
			IsUnique:     cast.ToInt(re["IS_UNIQUE"]) == 1,
			SeqInIndex:   cast.ToInt(re["SEQ_IN_INDEX"]),
			IsPrimaryKey: cast.ToInt(re["IS_PRIMARY"]) == 1,
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	return dbi.GroupIndexColumns(indexes), nil
}

// 获取建表ddl
func (od *OracleMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return dbi.GenTableDDL(od.di.GetDialect(), od, tableName, dropBeforeCreate)
}

// GetSchemas 获取当前连接的库可访问的 schemaNames。
//
// 12c+ 的 ALL_USERS 带 ORACLE_MAINTAINED 列（Y 表示由 catalog.sql 等 Oracle 脚本创建并维护），
// 据此隐藏 SYS/SYSTEM 等内置账号：它们不是业务 schema，在前端 schema 树里展开只会看到数据字典对象。
// 11g 没有该列（直接引用会 ORA-00904 让整个 schema 列表请求失败），故先探测列是否存在，
// 缺失时返回全量清单——多显示内置账号只是噪声，而让列表整体不可用会切断库树的 schema 层。
func (od *OracleMetadata) GetSchemas() ([]string, error) {
	sqlKey := ORACLE_DB_SCHEMAS
	if od.hasOracleMaintainedColumn() {
		sqlKey = ORACLE_DB_SCHEMAS_MAINTAINED
	}
	_, res, err := od.di.Query(metaSQL.Get(sqlKey))
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, cast.ToString(re["USERNAME"]))
	}
	return schemaNames, nil
}

// hasOracleMaintainedColumn 探测 12c+ 的 ALL_USERS.ORACLE_MAINTAINED 列是否存在。
// 探测失败按不存在处理：过滤器是可选优化，不能成为 schema 列表的可用性前提。
func (od *OracleMetadata) hasOracleMaintainedColumn() bool {
	_, res, err := od.di.Query("SELECT COUNT(*) AS CNT FROM SYS.ALL_TAB_COLUMNS" +
		" WHERE OWNER = 'SYS' AND TABLE_NAME = 'ALL_USERS' AND COLUMN_NAME = 'ORACLE_MAINTAINED'")
	if err != nil || len(res) == 0 {
		return false
	}
	return cast.ToInt(res[0]["CNT"]) > 0
}
