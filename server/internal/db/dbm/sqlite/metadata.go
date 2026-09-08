package sqlite

import (
	_ "embed"
	"errors"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"strings"

	"github.com/spf13/cast"
)

//go:embed meta.sql
var metaSqlFile string

// metaSql 方言元数据SQL模板（按备注key解析并缓存，格式见dbi.SqlTemplates）
var metaSql = dbi.NewSqlTemplates(metaSqlFile)

const (
	SQLITE_TABLE_INFO_KEY = "SQLITE_TABLE_INFO"
	SQLITE_INDEX_INFO_KEY = "SQLITE_INDEX_INFO"
)

var (
	// 提取声明类型的类型名与长度/精度参数：SQLite按DDL原文保存声明类型，必须容忍书写空白
	// （如 decimal(10, 2)、varchar (100)）：不允许空白时这些写法匹配不到，整串会被当作类型名，
	// 使列落入未注册类型的varchar兼容，结构迁移时数值/长度语义静默失真
	dataTypeRegexp = regexp.MustCompile(`(\w+)\s*\(\s*(\d*)\s*,?\s*(\d*)\s*\)`)
)

var _ dbi.Metadata = (*SqliteMetadata)(nil)

type SqliteMetadata struct {
	dbi.DefaultMetadata

	dc *dbi.DbConn
}

func (sd *SqliteMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := sd.dc.Query("SELECT SQLITE_VERSION() as version")
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

func (sd *SqliteMetadata) GetDbNames() ([]string, error) {
	databases := make([]string, 0)
	_, res, err := sd.dc.Query("PRAGMA database_list")
	if err != nil {
		return nil, err
	}
	for _, re := range res {
		databases = append(databases, cast.ToString(re["name"]))
	}

	return databases, nil
}

// 获取表基础元信息, 如表名等
func (sd *SqliteMetadata) GetTables(tableNames ...string) ([]dbi.Table, error) {
	dialect := sd.dc.GetDialect()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.QuoteEscape(dialect.Quoter().Trim(val)))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(metaSql.Get(SQLITE_TABLE_INFO_KEY), collx.M{"tableNames": names})
	if err != nil {
		return nil, err
	}

	_, res, err = sd.dc.Query(sql)
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

// GetDataTypes 正则提取字段类型中的关键字，
// 如 decimal(10,2)  提取decimal, 10 ,2
// 如:text 提取text,null,null
// 如:varchar(100)  提取varchar, 100
func (sd *SqliteMetadata) getDataTypes(dataType string) (string, string, string) {
	matches := dataTypeRegexp.FindStringSubmatch(dataType)
	if len(matches) == 0 {
		return dataType, "", ""
	}
	return matches[1], matches[2], matches[3]
}

// 获取列元信息, 如列名等
func (sd *SqliteMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	columns := make([]dbi.Column, 0)
	for i := 0; i < len(tableNames); i++ {
		tableName := tableNames[i]
		// 表名必须作为标识符引用后拼入PRAGMA：直接插值会使含空格/分号/引号/注释符的表名报语法错误；
		// 且失败必须返回错误而非continue，否则调用方（如dump）会拿到空列集生成无列的DDL/INSERT而静默丢数据
		_, res, err := sd.dc.Query(fmt.Sprintf("PRAGMA table_info(%s)", sd.dc.GetDialect().Quoter().QuoteIdent(tableName)))
		if err != nil {
			return nil, errorx.NewBizf("failed to get columns for table [%s]: %s", tableName, err.Error())
		}
		for _, re := range res {
			// 默认值保留 dflt_value 的书写原文（带引号的字面量形态），由SQLGenerator统一按字面量语义
			// 还原并重新转义；旧实现在此处ReplaceAll删除了内容中的全部单引号，使 DEFAULT 'it''s'
			// 的默认值失真为 its（结构迁移静默改变表定义）；若在此提前剥引号，则 '(0)' 这类
			// 内容本身含括号的默认值会与表达式默认值无法区分而被丢弃
			defaultValue := cast.ToString(re["dflt_value"])

			// 切割类型和长度，如果长度内有逗号，则说明是decimal类型
			columnType := cast.ToString(re["type"])
			dataType, length, scale := sd.getDataTypes(columnType)
			pkPos := cast.ToInt(re["pk"])

			column := dbi.Column{
				TableName:     tableName,
				ColumnName:    cast.ToString(re["name"]),
				ColumnComment: "",
				Nullable:      cast.ToInt(re["notnull"]) != 1,
				// pk是列在主键内的序号（0表示非主键列），复合主键的所有列都是主键；
				// 旧实现 pk==1 仅标记首个主键列，导致复合主键表迁移时其余主键列静默失去主键约束
				IsPrimaryKey: pkPos != 0,
				// 仅 INTEGER 主键是 rowid 别名而具备自增语义；声明为其他类型的主键列（如text主键）
				// 不应标记自增，否则DDL重建时会被强制改写为 integer 主键（列类型静默改变）
				AutoIncrement: pkPos == 1 && strings.EqualFold(dataType, "INTEGER"),
				ColumnDefault: defaultValue,
				NumScale:      0,
			}

			if scale != "0" && scale != "" {
				column.NumPrecision = cast.ToInt(length)
				column.NumScale = cast.ToInt(scale)
				column.CharMaxLength = 0
			} else {
				column.CharMaxLength = cast.ToInt(length)
			}
			column.DataType = strings.ToLower(dataType)

			sd.dc.GetDbDataType(column.DataType).FixColumn(&column)
			// sqlite的dflt_value是建表原文：字面量恒带引号，不带引号的函数调用/括号运算形态即表达式默认值
			dbi.MarkExprDefault(&column)
			columns = append(columns, column)
		}
	}
	return columns, nil
}

func (sd *SqliteMetadata) GetPrimaryKey(tableName string) (string, error) {
	_, res, err := sd.dc.Query(fmt.Sprintf("PRAGMA table_info(%s)", sd.dc.GetDialect().Quoter().QuoteIdent(tableName)))
	if err != nil {
		return "", err
	}
	for _, re := range res {
		if cast.ToInt(re["pk"]) == 1 {
			return cast.ToString(re["name"]), nil
		}
	}

	return "", errors.New("不存在主键")
}

// 解析索引创建语句以获取字段信息
func extractIndexFields(indexSQL string) string {
	// 使用正则表达式提取字段信息
	re := regexp.MustCompile(`\((.*?)\)`)
	match := re.FindStringSubmatch(indexSQL)
	if len(match) > 1 {
		fields := strings.Split(match[1], ",")
		for i, field := range fields {
			// 去除空格
			fields[i] = strings.TrimSpace(field)
		}
		return strings.Join(fields, ",")
	}
	return ""
}

// 获取表索引信息
func (sd *SqliteMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	// 模板以字符串字面量匹配表名，必须转义单引号，否则含单引号的表名会破坏SQL
	_, res, err := sd.dc.Query(fmt.Sprintf(metaSql.Get(SQLITE_INDEX_INFO_KEY), dbi.QuoteEscape(tableName)))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexSql := cast.ToString(re["indexSql"])
		isUnique := strings.Contains(indexSql, "CREATE UNIQUE INDEX")

		indexs = append(indexs, dbi.Index{
			IndexName:    cast.ToString(re["indexName"]),
			ColumnName:   extractIndexFields(indexSql),
			IndexType:    cast.ToString(re["indexType"]),
			IndexComment: cast.ToString(re["indexComment"]),
			IsUnique:     isUnique,
			SeqInIndex:   1,
			IsPrimaryKey: false,
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	return indexs, nil
}

// 获取建表ddl
func (sd *SqliteMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	var builder strings.Builder

	if dropBeforeCreate {
		builder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s; \n\n", sd.dc.GetDialect().Quoter().QuoteIdent(tableName)))
	}

	_, res, err := sd.dc.Query("select sql from sqlite_master WHERE tbl_name=? order by type desc", tableName)
	if err != nil {
		return "", err
	}

	for _, re := range res {
		builder.WriteString(cast.ToString(re["sql"]) + "; \n\n")
	}

	return builder.String(), nil
}

func (sd *SqliteMetadata) GetSchemas() ([]string, error) {
	return nil, nil
}
