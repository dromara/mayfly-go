package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"regexp"
	"strings"
	"time"
)

const (
	SQLITE_META_FILE      = "metasql/sqlite_meta.sql"
	SQLITE_TABLE_INFO_KEY = "SQLITE_TABLE_INFO"
	SQLITE_INDEX_INFO_KEY = "SQLITE_INDEX_INFO"
)

type SqliteDialect struct {
	dc *dbi.DbConn
}

func (sd *SqliteDialect) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := sd.dc.Query("SELECT SQLITE_VERSION() as version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["version"]),
	}
	return ds, nil
}

func (sd *SqliteDialect) GetDbNames() ([]string, error) {
	databases := make([]string, 0)
	_, res, err := sd.dc.Query("PRAGMA database_list")
	if err != nil {
		return nil, err
	}
	for _, re := range res {
		databases = append(databases, anyx.ConvString(re["name"]))
	}

	return databases, nil
}

// 获取表基础元信息, 如表名等
func (sd *SqliteDialect) GetTables() ([]dbi.Table, error) {
	_, res, err := sd.dc.Query(dbi.GetLocalSql(SQLITE_META_FILE, SQLITE_TABLE_INFO_KEY))
	//cols, res, err := sd.dc.Query("SELECT datetime(1092941466, 'unixepoch')")
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    anyx.ConvString(re["tableName"]),
			TableComment: anyx.ConvString(re["tableComment"]),
			CreateTime:   anyx.ConvString(re["createTime"]),
			TableRows:    anyx.ConvInt(re["tableRows"]),
			DataLength:   anyx.ConvInt64(re["dataLength"]),
			IndexLength:  anyx.ConvInt64(re["indexLength"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (sd *SqliteDialect) GetColumns(tableNames ...string) ([]dbi.Column, error) {

	columns := make([]dbi.Column, 0)

	for i := 0; i < len(tableNames); i++ {
		tableName := tableNames[i]
		_, res, err := sd.dc.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			logx.Error("获取数据库表字段结构出错", err.Error())
			continue
		}
		for _, re := range res {
			nullable := "YES"
			if anyx.ConvInt(re["notnull"]) == 1 {
				nullable = "NO"
			}
			// 去掉默认值的引号
			defaultValue := anyx.ConvString(re["dflt_value"])
			if strings.Contains(defaultValue, "'") {
				defaultValue = strings.ReplaceAll(defaultValue, "'", "")
			}
			columns = append(columns, dbi.Column{
				TableName:     tableName,
				ColumnName:    anyx.ConvString(re["name"]),
				ColumnType:    strings.ToLower(anyx.ConvString(re["type"])),
				ColumnComment: "",
				Nullable:      nullable,
				IsPrimaryKey:  anyx.ConvInt(re["pk"]) == 1,
				IsIdentity:    anyx.ConvInt(re["pk"]) == 1,
				ColumnDefault: defaultValue,
				NumScale:      "0",
			})
		}
	}
	return columns, nil
}

func (sd *SqliteDialect) GetPrimaryKey(tableName string) (string, error) {
	_, res, err := sd.dc.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return "", err
	}
	for _, re := range res {
		if anyx.ConvInt(re["pk"]) == 1 {
			return anyx.ConvString(re["name"]), nil
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
func (sd *SqliteDialect) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := sd.dc.Query(fmt.Sprintf(dbi.GetLocalSql(SQLITE_META_FILE, SQLITE_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexSql := anyx.ConvString(re["indexSql"])
		isUnique := strings.Contains(indexSql, "CREATE UNIQUE INDEX")

		indexs = append(indexs, dbi.Index{
			IndexName:    anyx.ConvString(re["indexName"]),
			ColumnName:   extractIndexFields(indexSql),
			IndexType:    anyx.ConvString(re["indexType"]),
			IndexComment: anyx.ConvString(re["indexComment"]),
			IsUnique:     isUnique,
			SeqInIndex:   1,
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	return indexs, nil
}

// 获取建表ddl
func (sd *SqliteDialect) GetTableDDL(tableName string) (string, error) {
	_, res, err := sd.dc.Query("select sql from sqlite_master WHERE tbl_name=? order by type desc", tableName)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	for _, re := range res {
		builder.WriteString(anyx.ConvString(re["sql"]) + "; \n\n")
	}

	return builder.String(), nil
}

func (sd *SqliteDialect) GetSchemas() ([]string, error) {
	return nil, nil
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (sd *SqliteDialect) GetDbProgram() (dbi.DbProgram, error) {
	return nil, fmt.Errorf("该数据库类型不支持数据库备份与恢复: %v", sd.dc.Info.Type)
}

func (sd *SqliteDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	// 执行批量insert sql，跟mysql一样 支持批量insert语法
	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	// 重复占位符字符串n遍
	repeated = strings.Repeat(placeholder+",", len(values))
	// 去除最后一个逗号
	placeholder = strings.TrimSuffix(repeated, ",")

	sqlStr := fmt.Sprintf("insert into %s (%s) values %s", sd.dc.Info.Type.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}

	// 执行批量insert sql
	return sd.dc.TxExec(tx, sqlStr, args...)
}

func (sd *SqliteDialect) GetDataConverter() dbi.DataConverter {
	return new(DataConverter)
}

var (
	// 数字类型
	numberRegexp = regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit|real`)
	// 日期时间类型
	datetimeRegexp = regexp.MustCompile(`(?i)datetime`)
)

type DataConverter struct {
}

func (dc *DataConverter) GetDataType(dbColumnType string) dbi.DataType {
	if numberRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	if datetimeRegexp.MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	return dbi.DataTypeString
}

func (dc *DataConverter) FormatData(dbColumnValue any, dataType dbi.DataType) string {
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

func (dc *DataConverter) ParseData(dbColumnValue any, dataType dbi.DataType) any {
	return dbColumnValue
}

func (sd *SqliteDialect) CopyTable(copy *dbi.DbCopyTable) error {
	tableName := copy.TableName

	// 生成新表名,为老表明+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")
	ddl, err := sd.GetTableDDL(tableName)
	if err != nil {
		return err
	}
	// 生成建表语句
	// 替换表名
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("CREATE TABLE \"%s\"", tableName), fmt.Sprintf("CREATE TABLE \"%s\"", newTableName))
	// 替换索引名，索引名为按照规范生成的，才能替换，否则未知索引名，无法替换
	ddl = strings.ReplaceAll(ddl, fmt.Sprintf("CREATE INDEX \"%s", tableName), fmt.Sprintf("CREATE INDEX \"%s", newTableName))

	// 执行建表语句
	_, err = sd.dc.Exec(ddl)
	if err != nil {
		return err
	}

	// 使用异步线程插入数据
	if copy.CopyData {
		go func() {
			// 执行插入语句
			_, _ = sd.dc.Exec(fmt.Sprintf("INSERT INTO \"%s\" SELECT * FROM \"%s\"", newTableName, tableName))
		}()
	}

	return err
}
