package mysql

import (
	_ "embed"
	"errors"
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
	MYSQL_DBS            = "MYSQL_DBS"
	MYSQL_TABLE_INFO_KEY = "MYSQL_TABLE_INFO"
	MYSQL_INDEX_INFO_KEY = "MYSQL_INDEX_INFO"
	MYSQL_COLUMN_MA_KEY  = "MYSQL_COLUMN_MA"
)

var _ dbi.Metadata = (*MysqlMetadata)(nil)

type MysqlMetadata struct {
	dbi.DefaultMetadata

	dc *dbi.DbConn
}

func (md *MysqlMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := md.dc.Query("SELECT VERSION() version")
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

func (md *MysqlMetadata) GetDbNames() ([]string, error) {
	_, res, err := md.dc.Query(metaSql.Get(MYSQL_DBS))
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, cast.ToString(re["dbname"]))
	}
	return databases, nil
}

func (md *MysqlMetadata) GetTables(tableNames ...string) ([]dbi.Table, error) {
	dialect := md.dc.GetDialect()
	names := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.QuoteEscapeBackslash(dialect.Quoter().Trim(val)))
	}), ",")

	var res []map[string]any
	var err error

	sql, err := stringx.TemplateParse(metaSql.Get(MYSQL_TABLE_INFO_KEY), collx.M{"tableNames": names})
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

// columnDefaultOf 归一列默认值的两种“空”形态与NULL歧义：
//  1. information_schema中「无默认值」是SQL NULL（扫描为nil），「空字符串默认值」是空字符串，
//     两者经cast.ToString后均为空串而无法区分，会使空串默认值在结构迁移时被当作无默认值静默丢弃；
//     故把空串归一为其字面量书写形态（一对单引号），由SQLGenerator按字面量语义统一处理。
//  2. MySQL 8.0起COLUMN_DEFAULT为去引号的原始值（已真实库实测）：DEFAULT NULL呈现为SQL NULL，
//     而字符串默认值'NULL'呈现为裸文本NULL；5.7及更早版本返回带引号字面量，裸文本NULL即DEFAULT NULL。
//     NOT NULL列不可能拥有DEFAULT NULL（建表即报Invalid default value，已实测），
//     故非空列上的裸NULL必定是字符串'NULL'，按字面量形态还原；可空列上的裸NULL保留无默认值语义以兼容5.7。
func columnDefaultOf(val any, nullable bool) string {
	if val == nil {
		return ""
	}
	if b, ok := val.([]byte); ok && b == nil {
		return ""
	}
	s := cast.ToString(val)
	if s == "" {
		return "''"
	}
	if s == "NULL" && !nullable {
		return "'NULL'"
	}
	return s
}

// 获取列元信息, 如列名等
func (md *MysqlMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dialect := md.dc.GetDialect()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.QuoteEscapeBackslash(dialect.Quoter().Trim(val)))
	}), ",")

	_, res, err := md.dc.Query(fmt.Sprintf(metaSql.Get(MYSQL_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		dataType := cast.ToString(re["dataType"])
		// information_schema的data_type不带unsigned后缀（如"int"），unsigned信息仅在column_type中；
		// 迁移链路靠dataType匹配注册类型，此处需归一化为"unsigned xxx"，
		// 否则无符号列跨库迁移时会退化为有符号类型，存在大值溢出/截断风险
		if strings.HasSuffix(strings.ToLower(cast.ToString(re["columnType"])), " unsigned") {
			dataType = "unsigned " + dataType
		}

		nullable := cast.ToString(re["nullable"]) == "YES"
		column := dbi.Column{
			TableName:     cast.ToString(re["tableName"]),
			ColumnName:    cast.ToString(re["columnName"]),
			ColumnType:    cast.ToString(re["columnType"]),
			DataType:      dataType,
			ColumnComment: cast.ToString(re["columnComment"]),
			Nullable:      nullable,
			IsPrimaryKey:  cast.ToInt(re["isPrimaryKey"]) == 1,
			AutoIncrement: cast.ToInt(re["autoIncrement"]) == 1,
			ColumnDefault: columnDefaultOf(re["columnDefault"], nullable),
			CharMaxLength: cast.ToInt(re["charMaxLength"]),
			NumPrecision:  cast.ToInt(re["numPrecision"]),
			NumScale:      cast.ToInt(re["numScale"]),
			IsExprDefault: cast.ToInt(re["isExprDefault"]) == 1,
			// 生成列（VIRTUAL/STORED GENERATED）：值由表达式派生，不可显式插入；派生表达式需另查
			// GENERATION_EXPRESSION（自MySQL 8.0.13才有，5.7/MariaDB无该列），故在列循环后补取
			IsGenerated: cast.ToInt(re["isGenerated"]) == 1,
		}
		// 自动更新子句（on update CURRENT_TIMESTAMP[(fsp)]）仅存于EXTRA，以扩展信息承载，
		// 由MySQL生成器按目标列fsp写回DDL（其他方言无该语法，不会读取此信息）
		if onUpdate := cast.ToString(re["onUpdate"]); onUpdate != "" {
			column.Extra = collx.M{dbi.ColumnExtraOnUpdate: onUpdate}
		}

		md.dc.GetDbDataType(column.DataType).FixColumn(&column)
		columns = append(columns, column)
	}

	// 补取生成列的派生表达式（仅存在生成列时才会发起）
	md.markGeneratedColumnExpr(columns)
	return columns, nil
}

// mysqlGenColumn SHOW CREATE TABLE中解析出的一条生成列定义：表达式原文与是否物化存储

type mysqlGenColumn struct {
	expr   string
	stored bool
}

// markGeneratedColumnExpr 补取生成列的派生表达式与物化方式（仅表内确实存在生成列时才发起查询）。
//
// 表达式必须取自SHOW CREATE TABLE而不是information_schema.COLUMNS.GENERATION_EXPRESSION，
// 实测（MySQL 8.0.46）后者有三处会使重建DDL失真的问题：
//  1. 整个表达式文本又被按MySQL字符串字面量规则多转义了一层，concat(`a`,'-')回报为
//     concat(`a`,_utf8mb4\'-\')（HEX证实为5C 27），原文嵌入DDL必然语法错误；
//  2. 表达式的值按latin1解释后再转成连接字符集，含非ASCII字面量时双重编码，'中文'的字节
//     由E4B8AD变成C3A4C2B8AD，重建后该列值静默乱码（比报错更隐蔽）；
//  3. 旧版本存在长度截断，截断后的表达式派生语义已改变。
//
// SHOW CREATE TABLE是MySQL自身给出的可原样重放的DDL文本（mysqldump亦以其为源），三项均有保证。
// 解析不到该列（表达式含裸换行等非常形态）则不标记，退回「目标建普通列 + 插入源值」的保守语义，
// 绝不拿残缺或形态陌生的文本去猜DDL
func (md *MysqlMetadata) markGeneratedColumnExpr(columns []dbi.Column) {
	// 按表归集生成列下标，无生成列则完全不发起额外查询
	tableColumnIdx := make(map[string][]int)
	for i, col := range columns {
		if col.IsGenerated {
			tableColumnIdx[col.TableName] = append(tableColumnIdx[col.TableName], i)
		}
	}
	if len(tableColumnIdx) == 0 {
		return
	}

	quoter := md.dc.GetDialect().Quoter()
	for tableName, idxs := range tableColumnIdx {
		_, res, err := md.dc.Query("SHOW CREATE TABLE " + quoter.QuoteIdent(tableName))
		if err != nil || len(res) == 0 {
			continue
		}
		for _, idx := range idxs {
			column := &columns[idx]
			genColumn, ok := parseMysqlGeneratedColumns(mysqlShowCreateText(res[0]))[strings.ToLower(column.ColumnName)]
			if !ok {
				continue
			}
			kind := dbi.GenerationVirtual
			if genColumn.stored {
				kind = dbi.GenerationStored
			}
			dbi.MarkGeneratedColumn(column, string(DbTypeMysql), genColumn.expr, kind)
		}
	}
}

// mysqlShowCreateText 取SHOW CREATE TABLE结果中的建表语句文本。
// 结果集列名为「Create Table」（含空格），不同版本/MariaDB大小写可能不同，故按大小写不敏感匹配，
// 且必须排除MariaDB额外的「Create View」等列
func mysqlShowCreateText(row map[string]any) string {
	for key, val := range row {
		if strings.EqualFold(key, "Create Table") {
			return cast.ToString(val)
		}
	}
	return ""
}

// parseMysqlGeneratedColumns 解析SHOW CREATE TABLE文本中各生成列的「AS (expr) {VIRTUAL|STORED}」定义，
// 返回以（小写）列名为键的映射。
//
// MySQL的生成列定义恒为独立一行且列名必以反引号引用，物化关键字紧随表达式之后，
// 因此只认这一形态；其余行（普通列、索引、约束、表选项）一律跳过
func parseMysqlGeneratedColumns(createSql string) map[string]mysqlGenColumn {
	genColumns := make(map[string]mysqlGenColumn)
	for _, line := range strings.Split(createSql, "\n") {
		line = strings.TrimRight(strings.TrimSpace(line), ",")
		if !strings.HasPrefix(line, "`") {
			continue
		}
		nameEnd := strings.Index(line[1:], "`")
		if nameEnd < 0 {
			continue
		}
		rest := line[nameEnd+2:]
		asIdx := strings.Index(strings.ToUpper(rest), generatedAlwaysAsKeyword)
		if asIdx < 0 {
			continue
		}
		expr, tail, ok := splitMysqlParenContent(strings.TrimSpace(rest[asIdx+len(generatedAlwaysAsKeyword):]))
		if !ok || strings.TrimSpace(expr) == "" {
			continue
		}
		// 物化关键字必须紧随表达式（可能再跟NOT NULL/COMMENT等），否则形态未知，不重建
		tail = strings.ToUpper(strings.TrimSpace(tail))
		stored := strings.HasPrefix(tail, "STORED") || strings.HasPrefix(tail, "PERSISTENT")
		if !stored && !strings.HasPrefix(tail, "VIRTUAL") {
			continue
		}
		// 反引号标识符内的反引号以双写转义，还原后才能与元数据列名对上
		name := strings.ReplaceAll(line[1:nameEnd+1], "``", "`")
		genColumns[strings.ToLower(name)] = mysqlGenColumn{expr: expr, stored: stored}
	}
	return genColumns
}

// generatedAlwaysAsKeyword 生成列定义的固定引导词（大小写不敏感匹配）
const generatedAlwaysAsKeyword = "GENERATED ALWAYS AS"

// splitMysqlParenContent 切分出以「(」开头的文本的最外层括号内容与其余部分。
//
// 必须整体跳过字符串字面量与反引号标识符：括号与引号可在字面量内任意出现（如concat(a,'(')），
// 按计数法切割会在错误位置闭合；字面量未闭合（表达式内含裸换行使行被截断）时判定失败
func splitMysqlParenContent(s string) (inner, rest string, ok bool) {
	if len(s) == 0 || s[0] != '(' {
		return "", "", false
	}
	depth := 0
	for i := 0; i < len(s); {
		switch s[i] {
		case '\'', '"':
			end, ok := skipMysqlQuoted(s, i, s[i], true)
			if !ok {
				return "", "", false
			}
			i = end
			continue
		case '`':
			end, ok := skipMysqlQuoted(s, i, '`', false)
			if !ok {
				return "", "", false
			}
			i = end
			continue
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[1:i], s[i+1:], true
			}
		}
		i++
	}
	return "", "", false
}

// skipMysqlQuoted 跳过s[start]处开始的一段引用内容，返回其结束位置的下一索引。
// quote为引用符，backslashEscape为true时（字符串字面量）反斜杠转义下一个字符，
// 否则（反引号标识符）反斜杠属于名称本身；同名引用符双写表示转义
func skipMysqlQuoted(s string, start int, quote byte, backslashEscape bool) (int, bool) {
	for i := start + 1; i < len(s); {
		if backslashEscape && s[i] == '\\' {
			i += 2
			continue
		}
		if s[i] == quote {
			if i+1 < len(s) && s[i+1] == quote {
				i += 2
				continue
			}
			return i + 1, true
		}
		i++
	}
	return 0, false
}

// 获取表主键字段名，不存在主键标识则默认第一个字段
func (md *MysqlMetadata) GetPrimaryKey(tablename string) (string, error) {
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

// 获取表索引信息
func (md *MysqlMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := md.dc.Query(metaSql.Get(MYSQL_INDEX_INFO_KEY), tableName)
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
			Extra:        collx.Kvs(IndexSubPartKey, cast.ToInt(re[IndexSubPartKey])),
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
func (md *MysqlMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return dbi.GenTableDDL(md.dc.GetDialect(), md, tableName, dropBeforeCreate)
}

func (md *MysqlMetadata) GetSchemas() ([]string, error) {
	return nil, errors.New("不支持schema")
}
