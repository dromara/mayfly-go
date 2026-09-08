package postgres

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
var metaSqlFile string

// metaSql 方言元数据SQL模板（按备注key解析并缓存，格式见dbi.SqlTemplates）
var metaSql = dbi.NewSqlTemplates(metaSqlFile)

const (
	PGSQL_DB_SCHEMAS     = "PGSQL_DB_SCHEMAS"
	PGSQL_TABLE_INFO_KEY = "PGSQL_TABLE_INFO"
	PGSQL_INDEX_INFO_KEY = "PGSQL_INDEX_INFO"
	PGSQL_COLUMN_MA_KEY  = "PGSQL_COLUMN_MA"
)

var _ dbi.Metadata = (*PgsqlMetadata)(nil)

type PgsqlMetadata struct {
	dbi.DefaultMetadata

	dc *dbi.DbConn
}

func (pd *PgsqlMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := pd.dc.Query("SELECT version() as server_version")
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errorx.NewBiz("failed to get database version: empty result")
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
		return fmt.Sprintf("'%s'", dbi.QuoteEscape(dialect.Quoter().Trim(val)))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(metaSql.Get(PGSQL_TABLE_INFO_KEY), collx.M{"tableNames": names})
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
		return fmt.Sprintf("'%s'", dbi.QuoteEscape(dialect.Quoter().Trim(val)))
	}), ",")

	_, res, err := pd.dc.Query(fmt.Sprintf(metaSql.Get(PGSQL_COLUMN_MA_KEY), tableName))
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

	// 补记information_schema无法呈现的两类列语义（生成列/标识列），详见函数注释
	pd.markGeneratedColumns(tableName, columns)
	return columns, nil
}

// markGeneratedColumns 从系统目录补列信息_schema完全看不出来：
//   - 生成列（attgenerated为's'存储/'v'虚拟）：column_default为空，但向生成列显式插入值会直接报错（3402），
//     会使同构pg->pg整表数据迁移/同步失败，必须识别后从INSERT剔除；派生表达式取自其默认值定义（pg_attrdef）
//   - 标识列（attidentity为'a'/'d'，pg 10+的GENERATED ALWAYS AS IDENTITY）：column_default也为空，
//     仅靠nextval无法识别，不标记则自增列静默退化为普通整型列（目标库后续插入不再自动取值）
//
// attgenerated/attidentity 自 pg 12/pg 10 引入，GaussDB/Vastbase/Kingbase 等基于 pg 9.x 的兼容库可能无这些列，
// 若并入主列元数据查询会使整个列信息查询直接报错（影响远大于收益），故单独查询并在失败时静默跳过
func (pd *PgsqlMetadata) markGeneratedColumns(tableNamesLiteral string, columns []dbi.Column) {
	sql := fmt.Sprintf(`SELECT c.relname AS "tableName", a.attname AS "columnName", a.attgenerated::text AS "kind", `+
		`a.attidentity::text AS "identity", pg_get_expr(d.adbin, d.adrelid) AS "expr" FROM pg_attribute a `+
		`JOIN pg_class c ON c.oid = a.attrelid `+
		`LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum `+
		`WHERE c.relname IN (%s) AND a.attnum > 0 AND (a.attgenerated <> '' OR a.attidentity <> '')`, tableNamesLiteral)
	_, res, err := pd.dc.Query(sql)
	if err != nil || len(res) == 0 {
		return
	}
	columnIdx := make(map[string]int, len(columns))
	for i, col := range columns {
		columnIdx[col.TableName+"."+col.ColumnName] = i
	}
	// 标识列属于序列类默认值（主键上可安全保留自增语义），其元数据不呈现取值表达式，需单独标记不记录表达式
	srcDbType := cast.ToString(pd.dc.Info.Type)
	for _, re := range res {
		idx, ok := columnIdx[cast.ToString(re["tableName"])+"."+cast.ToString(re["columnName"])]
		if !ok {
			continue
		}
		column := &columns[idx]
		if identity := cast.ToString(re["identity"]); identity == "a" || identity == "d" {
			// 仅主键列标记自增：非主键的自增列迁入MySQL会因「自增列必须是键」直接建表失败，
			// 不标记仅丢失自动取值语义（存量数据仍按源值完整插入）
			if column.IsPrimaryKey {
				column.AutoIncrement = true
			}
			continue
		}
		// 无表达式可取时不记录（不重建也不剔除），否则目标列退化为普通列又不赋值会造成静默NULL
		expr := strings.TrimSpace(cast.ToString(re["expr"]))
		if expr == "" {
			continue
		}
		// 仅物化存储（'s'）的列才记录：VIRTUAL 生成列自 pg 18 才存在，目标库不支持时建表即失败
		if cast.ToString(re["kind"]) != "s" {
			continue
		}
		dbi.MarkGeneratedColumn(column, srcDbType, expr, dbi.GenerationStored)
	}
}

func (pd *PgsqlMetadata) GetPrimaryKey(tablename string) (string, error) {
	columns, err := pd.GetColumns(tablename)
	if err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", errorx.NewBizf("[%s] 表不存在", tablename)
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
	dialect := pd.dc.GetDialect()
	// 表名需转义单引号后拼入模板（模板为字符串字面量拼接，与GetColumns保持一致）
	escTableName := dbi.QuoteEscape(dialect.Quoter().Trim(tableName))
	_, res, err := pd.dc.Query(fmt.Sprintf(metaSql.Get(PGSQL_INDEX_INFO_KEY), escTableName))
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
	sql := metaSql.Get(PGSQL_DB_SCHEMAS)
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
