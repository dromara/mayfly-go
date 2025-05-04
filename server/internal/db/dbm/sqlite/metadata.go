package sqlite

import (
	"errors"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"strings"

	"github.com/may-fly/cast"
)

const (
	SQLITE_META_FILE      = "metasql/sqlite_meta.sql"
	SQLITE_TABLE_INFO_KEY = "SQLITE_TABLE_INFO"
	SQLITE_INDEX_INFO_KEY = "SQLITE_INDEX_INFO"
)

var (
	dataTypeRegexp = regexp.MustCompile(`(\w+)\((\d*),?(\d*)\)`)
)

type SqliteMetadata struct {
	dbi.DefaultMetadata

	dc *dbi.DbConn
}

func (sd *SqliteMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := sd.dc.Query("SELECT SQLITE_VERSION() as version")
	if err != nil {
		return nil, err
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
		return fmt.Sprintf("'%s'", dialect.Quoter().Trim(val))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(dbi.GetLocalSql(SQLITE_META_FILE, SQLITE_TABLE_INFO_KEY), collx.M{"tableNames": names})
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
		_, res, err := sd.dc.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			logx.Error("获取数据库表字段结构出错", err.Error())
			continue
		}
		for _, re := range res {
			// 去掉默认值的引号
			defaultValue := cast.ToString(re["dflt_value"])
			if strings.Contains(defaultValue, "'") {
				defaultValue = strings.ReplaceAll(defaultValue, "'", "")
			}

			column := dbi.Column{
				TableName:     tableName,
				ColumnName:    cast.ToString(re["name"]),
				ColumnComment: "",
				Nullable:      cast.ToInt(re["notnull"]) != 1,
				IsPrimaryKey:  cast.ToInt(re["pk"]) == 1,
				AutoIncrement: cast.ToInt(re["pk"]) == 1,
				ColumnDefault: defaultValue,
				NumScale:      0,
			}

			// 切割类型和长度，如果长度内有逗号，则说明是decimal类型
			columnType := cast.ToString(re["type"])
			dataType, length, scale := sd.getDataTypes(columnType)
			if scale != "0" && scale != "" {
				column.NumPrecision = cast.ToInt(length)
				column.NumScale = cast.ToInt(scale)
				column.CharMaxLength = 0
			} else {
				column.CharMaxLength = cast.ToInt(length)
			}
			column.DataType = strings.ToLower(dataType)

			sd.dc.GetDbDataType(column.DataType).FixColumn(&column)
			columns = append(columns, column)
		}
	}
	return columns, nil
}

func (sd *SqliteMetadata) GetPrimaryKey(tableName string) (string, error) {
	_, res, err := sd.dc.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
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
	_, res, err := sd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(SQLITE_META_FILE, SQLITE_INDEX_INFO_KEY), tableName))
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
		builder.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s; \n\n", tableName))
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
