package mysql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/spf13/cast"
)

var _ dbi.SQLGenerator = (*SQLGenerator)(nil)

type SQLGenerator struct {
	Dialect dbi.Dialect
}

func (msg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	sqlArr := make([]string, 0)
	quoter := msg.Dialect.Quoter()

	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", quoter.QuoteIdent(table.TableName)))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", quoter.QuoteIdent(table.TableName))
	fields := make([]string, 0)
	pks := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			// 主键列名同样是元数据返回的完整标识符（可含空格、分号等），必须引用，
			// 否则 PRIMARY KEY (id 主键;号) 这类语句直接语法错误
			pks = append(pks, quoter.QuoteIdent(column.ColumnName))
		}
		fields = append(fields, msg.genColumnBasicSql(quoter, column))
	}

	// 建表ddl
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	// 表注释
	if table.TableComment != "" {
		createSql += fmt.Sprintf(" COMMENT '%s'", dbi.QuoteEscapeBackslash(table.TableComment))
	}

	sqlArr = append(sqlArr, createSql)

	return sqlArr
}

func (msg *SQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	sqlArr := make([]string, 0)
	quoter := msg.Dialect.Quoter()

	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}
		// 取出列名，逐个按标识符引用：不能复用Quotes（其Quote会按空格/点切分片段，
		// 含空格的真实列名会被切成两段而生成非法DDL）
		colNames := make([]string, 0)
		for _, name := range strings.Split(index.ColumnName, ",") {
			colNames = append(colNames, quoter.QuoteIdent(name))
		}

		// 暂时先处理单个索引的情况，多个涉及获取索引时的合并等，以及前端调整等，后续完善
		if subPart := cast.ToInt(index.Extra[IndexSubPartKey]); subPart > 0 && len(colNames) == 1 {
			colNames[0] = fmt.Sprintf("%s(%d)", colNames[0], subPart)
		}

		sqlTmp := "ALTER TABLE %s ADD %s INDEX %s(%s) USING BTREE"
		sqlStr := fmt.Sprintf(sqlTmp, quoter.QuoteIdent(table.TableName), unique, quoter.QuoteIdent(index.IndexName), strings.Join(colNames, ","))
		comment := dbi.QuoteEscapeBackslash(index.IndexComment)
		if comment != "" {
			sqlStr += fmt.Sprintf(" COMMENT '%s'", comment)
		}
		sqlArr = append(sqlArr, sqlStr)
	}

	return sqlArr
}

func (msg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {
	if duplicateStrategy == dbi.DuplicateStrategyNone {
		return collx.AsArray(dbi.GenCommonInsert(msg.Dialect, DbTypeMysql, tableName, columns, values))
	}

	prefix := "insert ignore into"
	if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		prefix = "replace into"
	}

	quote := msg.Dialect.Quoter().QuoteIdent
	columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(msg.Dialect, DbTypeMysql, columns, values)

	return collx.AsArray[string](fmt.Sprintf("%s %s %s VALUES \n%s", prefix, quote(tableName), columnStr, strings.Join(valuesStrs, ",\n")))
}

func (msg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	dataType := column.DataType

	incr := ""
	if column.AutoIncrement {
		incr = " AUTO_INCREMENT"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}
	columnType := column.GetColumnType()
	// 注册类型名为"unsigned xxx"（与元数据data_type归一化对齐），但mysql DDL语法为"xxx unsigned"，
	// 需转为后缀形式，否则异构迁移生成DDL会报语法错误
	if strings.HasPrefix(columnType, "unsigned ") {
		columnType = strings.TrimPrefix(columnType, "unsigned ") + " unsigned"
	}
	// mysql的varchar/char/varbinary必须声明长度，无长度DDL直接语法错误；
	// 跨方言迁移时源列可能无长度信息（如sqlite numeric动态类型转varchar），补默认长度
	if !strings.Contains(columnType, "(") {
		lower := strings.ToLower(columnType)
		if strings.HasPrefix(lower, "varchar") || strings.HasPrefix(lower, "char") || strings.HasPrefix(lower, "varbinary") {
			columnType = fmt.Sprintf("%s(255)", columnType)
		}
	}
	if nullAble == "" && strings.Contains(columnType, "timestamp") {
		nullAble = " NULL"
	}

	// 默认值的呈现形态随MySQL版本而异（8.0为原始值，5.7/MariaDB为带引号字面量），
	// 且enum/set/binary等类型的默认值必须引用，统一由dbi按字面量/裸值语义还原并重新转义；
	// 日期时间类的自动初始化默认值受MySQL严格语法约束，必须先走mysqlTimeDefaultSql局部判定
	defVal, handled := mysqlTimeDefaultSql(column.ColumnDefault, columnType)
	if !handled {
		defVal = dbi.GenColumnDefaultSqlOf(&column, dataType, dbi.QuoteEscapeBackslash)
	}
	// BLOB/TEXT/JSON（含异构迁移过来的clob等大字列）在MySQL中不接受字面量默认值（报Error 1101），
	// 必须改写为8.0.13+的表达式默认值形态 DEFAULT ('xxx')，否则跨库建表直接失败
	if defVal != "" && mysqlNoLiteralDefaultType(dataType) {
		defVal = " DEFAULT (" + strings.TrimPrefix(defVal, " DEFAULT ") + ")"
	}
	comment := ""
	if column.ColumnComment != "" {
		// 防止注释内含有特殊字符串导致sql出错
		commentStr := dbi.QuoteEscapeBackslash(column.ColumnComment)
		comment = fmt.Sprintf(" COMMENT '%s'", commentStr)
	}

	// 生成列的DDL形态与普通列完全不同（无DEFAULT/AUTO_INCREMENT/ON UPDATE子句），单独成句：
	// 实测MySQL自己的SHOW CREATE形态为「`c` int GENERATED ALWAYS AS ((`a` + `b`)) STORED NOT NULL」，
	// 物化关键字必须紧跟表达式、NOT NULL再其后（两者颠倒即语法错误），注释最后；
	// 不重建则目标列退化为普通列，与INSERT阶段的剔除叠加会使生成列值静默变NULL
	if dbi.PreservableGeneratedColumn(column, DbTypeMysql) {
		storage := "VIRTUAL"
		if dbi.GeneratedColumnStored(column) {
			storage = "STORED"
		}
		genNull := ""
		if !column.Nullable {
			genNull = " NOT NULL"
		}
		return fmt.Sprintf(" %s %s GENERATED ALWAYS AS (%s) %s%s%s",
			quoter.QuoteIdent(column.ColumnName), columnType, dbi.GeneratedColumnExpr(column), storage, genNull, comment)
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s%s%s", quoter.QuoteIdent(column.ColumnName), columnType, nullAble, incr, defVal, mysqlOnUpdateSql(column, columnType), comment)
	return columnSql
}

// mysqlOnUpdateSql 生成MySQL的「自动更新」子句（ON UPDATE CURRENT_TIMESTAMP[(fsp)]），子句原文来自源列EXTRA。
//
// MySQL要求其小数秒与列定义的fsp严格一致（datetime(3) ON UPDATE CURRENT_TIMESTAMP 直接报Error 1067），
// 故不按原文直接回拼而是按目标列fsp重写；仅datetime/timestamp类列接受该子句，且只接受
// 「当前日期时间」关键字形态，其他内容（异常元数据）一律不写入，避免非法DDL与注入
func mysqlOnUpdateSql(column dbi.Column, columnType string) string {
	raw, _ := column.Extra[dbi.ColumnExtraOnUpdate].(string)
	expr := strings.TrimSpace(raw)
	if expr == "" {
		return ""
	}
	if lower := strings.ToLower(expr); strings.HasPrefix(lower, "on update ") {
		expr = strings.TrimSpace(expr[len("on update "):])
	}
	_, parts, _, ok := dbi.ParseTimeKeywordDefault(expr)
	if !ok || parts != dbi.TimePartDate|dbi.TimePartTime {
		return ""
	}
	base, fsp, _ := dbi.SplitColumnTypeBase(columnType)
	if !strings.Contains(base, "datetime") && !strings.Contains(base, "timestamp") {
		return ""
	}
	if fsp > 0 {
		if fsp > 6 {
			fsp = 6
		}
		return fmt.Sprintf(" ON UPDATE CURRENT_TIMESTAMP(%d)", fsp)
	}
	return " ON UPDATE CURRENT_TIMESTAMP"
}

// mysqlNoLiteralDefaultType 判断列类型是否属于MySQL不允许字面量默认值的大字列类型族
// （BLOB/TEXT/JSON，含text的longtext/mediumtext/tinytext与json均在内）
func mysqlNoLiteralDefaultType(dataType string) bool {
	lower := strings.ToLower(dataType)
	return strings.Contains(lower, "text") || strings.Contains(lower, "blob") || strings.Contains(lower, "json")
}

// mysqlTimeDefaultSql 生成MySQL日期时间类列的「当前日期/时间」默认值子句（含前导空格）。
//
// MySQL对该类默认值的语法约束远比通用判定严格（以下均探测自MySQL 8.0）：
//   - DATETIME/TIMESTAMP只接受CURRENT_TIMESTAMP（NOW/LOCALTIMESTAMP等同义写法归一为该形式）作为自动初始化
//     默认值，且其小数秒参数必须与列定义的fsp完全一致：datetime(6) DEFAULT CURRENT_TIMESTAMP 与
//     datetime DEFAULT CURRENT_TIMESTAMP(3) 都直接报Error 1067，故必须按目标列fsp书写；
//   - CURRENT_DATE/CURRENT_TIME/curdate()/SYSDATE等不是合法的裸默认值（Error 1064），
//     必须改写成8.0.13+的表达式默认值形态 (CURRENT_DATE)，否则异构迁移建表直接失败；
//   - 目标列只有日期/只有时间时，按目标列可承载的分量归一（如源CURRENT_TIMESTAMP迁入date列即取当日）
//
// handled为false表示该默认值不属于本函数处理范围（字面量/NULL/其他表达式），交由通用判定生成
func mysqlTimeDefaultSql(rawDefault string, columnType string) (defVal string, handled bool) {
	if rawDefault == "" {
		return "", false
	}
	// 部分元数据实现会将表达式默认值整体括号包裹后呈现（如(CURRENT_DATE)），先剥一层再识别
	if inner, balanced := dbi.UnwrapOuterParens(rawDefault); balanced {
		rawDefault = inner
	}
	_, parts, _, ok := dbi.ParseTimeKeywordDefault(rawDefault)
	if !ok {
		return "", false
	}
	baseType, fsp, _ := dbi.SplitColumnTypeBase(columnType)
	if !dbi.IsDateTimeType(baseType) {
		return "", false
	}
	switch dbi.TimePartsOfBaseType(baseType) {
	case dbi.TimePartDate:
		return " DEFAULT (CURRENT_DATE)", true
	case dbi.TimePartTime:
		return " DEFAULT (CURRENT_TIME)", true
	}
	// 目标为DATETIME/TIMESTAMP：纯日期/纯时间的默认值只能以表达式默认值形态书写
	switch parts {
	case dbi.TimePartDate:
		return " DEFAULT (CURRENT_DATE)", true
	case dbi.TimePartTime:
		return " DEFAULT (CURRENT_TIME)", true
	}
	if fsp > 0 {
		// 日期时间合一的写法（now()/LOCALTIMESTAMP/SYSTIMESTAMP等）统一归一为MySQL的CURRENT_TIMESTAMP：
		// 它们才是MySQL DATETIME/TIMESTAMP接受的自动初始化子，裸写源库函数名会直接语法错误
		if fsp > 6 {
			fsp = 6
		}
		return fmt.Sprintf(" DEFAULT CURRENT_TIMESTAMP(%d)", fsp), true
	}
	return " DEFAULT CURRENT_TIMESTAMP", true
}
