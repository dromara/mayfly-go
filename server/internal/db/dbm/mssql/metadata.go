package mssql

import (
	_ "embed"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
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
	MSSQL_DBS_KEY        = "MSSQL_DBS"
	MSSQL_DB_SCHEMAS_KEY = "MSSQL_DB_SCHEMAS"
	MSSQL_TABLE_INFO_KEY = "MSSQL_TABLE_INFO"
	MSSQL_INDEX_INFO_KEY = "MSSQL_INDEX_INFO"
	MSSQL_COLUMN_MA_KEY  = "MSSQL_COLUMN_MA"
)

var _ dbi.Metadata = (*MssqlMetadata)(nil)

type MssqlMetadata struct {
	dbi.DefaultMetadata

	dc *dbi.DbConn
}

func (md *MssqlMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := md.dc.Query("SELECT @@VERSION as version")
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errorx.NewBiz("failed to get database version: empty result")
	}
	ds := &dbi.DbServer{
		Version: cast.ToString(res[0]["version"]),
	}
	return ds, nil
}

func (md *MssqlMetadata) GetDbNames() ([]string, error) {
	_, res, err := md.dc.Query(metaSql.Get(MSSQL_DBS_KEY))
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, cast.ToString(re["dbname"]))
	}

	return databases, nil
}

// 获取表基础元信息, 如表名等
func (md *MssqlMetadata) GetTables(tableNames ...string) ([]dbi.Table, error) {
	dialect := md.dc.GetDialect()
	schema := md.dc.Info.CurrentSchema()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.QuoteEscape(dialect.Quoter().Trim(val)))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(metaSql.Get(MSSQL_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = md.dc.Query(sql, schema)
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
func (md *MssqlMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dialect := md.dc.GetDialect()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.QuoteEscape(dialect.Quoter().Trim(val)))
	}), ",")

	_, res, err := md.dc.Query(fmt.Sprintf(metaSql.Get(MSSQL_COLUMN_MA_KEY), tableName), md.dc.Info.CurrentSchema())
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		// (max)形态还原与nchar/nvarchar字节→字符换算统一由normalizeMssqlColumnType处理（顺序敏感，见其注释）
		dataType, charMaxLength := normalizeMssqlColumnType(anyx.ToString(re["DATA_TYPE"]), cast.ToInt(re["CHAR_MAX_LENGTH"]))

		column := dbi.Column{
			TableName:     anyx.ToString(re["TABLE_NAME"]),
			ColumnName:    anyx.ToString(re["COLUMN_NAME"]),
			DataType:      dataType,
			CharMaxLength: charMaxLength,
			ColumnComment: anyx.ToString(re["COLUMN_COMMENT"]),
			Nullable:      anyx.ToString(re["NULLABLE"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["IS_PRIMARY_KEY"]) == 1,
			AutoIncrement: cast.ToInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: cast.ToString(re["COLUMN_DEFAULT"]),
			NumPrecision:  cast.ToInt(re["NUM_PRECISION"]),
			NumScale:      cast.ToInt(re["NUM_SCALE"]),
			IsGenerated:   cast.ToInt(re["IS_GENERATED"]) == 1,
		}

		md.dc.GetDbDataType(column.DataType).FixColumn(&column)
		// SQL Server的字面量默认值在object_definition里恒带引号（('abc')、(N'abc')），
		// 剥括号后仍非字面量的（(getdate())、(suser_sname())）即表达式默认值，标记以免写成字符串
		dbi.MarkExprDefault(&column)
		// 计算列必须连定义原文一起标记：无定义则无法在目标库重建，按普通列建表并插入源值（保留数据）；
		// 有定义且目标同为SQL Server时才重建为计算列并从INSERT剔除，两者判定必须一致否则静默变NULL
		if column.IsGenerated {
			if def := strings.TrimSpace(cast.ToString(re["COMPUTED_DEF"])); def != "" {
				kind := dbi.GenerationVirtual
				if cast.ToInt(re["IS_PERSISTED"]) == 1 {
					kind = dbi.GenerationStored
				}
				dbi.MarkGeneratedColumn(&column, string(DbTypeMssql), def, kind)
			}
		}
		columns = append(columns, column)
	}
	return columns, nil
}

// 获取表主键字段名，不存在主键标识则默认第一个字段
func (md *MssqlMetadata) GetPrimaryKey(tablename string) (string, error) {
	columns, err := md.GetColumns(tablename)
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

// 需要收集唯一键涉及的字段，所以需要查询出带主键的索引
func (md *MssqlMetadata) getTableIndexWithPK(tableName string) ([]dbi.Index, error) {
	_, res, err := md.dc.Query(metaSql.Get(MSSQL_INDEX_INFO_KEY), md.dc.Info.CurrentSchema(), tableName)
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
	// 把查询结果以索引名分组，多个索引字段以逗号连接
	result := make([]dbi.Index, 0)
	key := ""
	for _, v := range indexs {
		// 当前的索引名
		in := v.IndexName
		if key == in {
			// 索引字段已根据名称和字段顺序排序，故取最后一个即可
			i := len(result) - 1
			// 同索引字段以逗号连接
			result[i].ColumnName = result[i].ColumnName + "," + v.ColumnName
		} else {
			key = in
			result = append(result, v)
		}
	}
	// 返回分组合并后的结果，原实现误返回未分组的indexs导致组合索引被拆为多行
	return result, nil
}

// 获取表索引信息
func (md *MssqlMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	indexs, _ := md.getTableIndexWithPK(tableName)
	result := make([]dbi.Index, 0)
	// 过滤掉主键索引
	for _, v := range indexs {
		if v.IsPrimaryKey {
			continue
		}
		result = append(result, v)
	}
	return result, nil
}

// 获取建表ddl
func (md *MssqlMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return dbi.GenTableDDL(md.dc.GetDialect(), md, tableName, dropBeforeCreate)
}

func (md *MssqlMetadata) GetSchemas() ([]string, error) {
	_, res, err := md.dc.Query(metaSql.Get(MSSQL_DB_SCHEMAS_KEY))
	if err != nil {
		return nil, err
	}

	schemas := make([]string, 0)
	for _, re := range res {
		schemas = append(schemas, cast.ToString(re["SCHEMA_NAME"]))
	}
	return schemas, nil
}
