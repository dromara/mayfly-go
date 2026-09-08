package dbi

import (
	"strings"
)

// 本文件为「列默认值」主题：解析各源库元数据呈现的默认值原文（形态互不相同），
// 并按目标方言/目标列类型生成DEFAULT子句。识别逻辑为形态驱动而非方言驱动，
// 详见 GenColumnDefaultSql 的决策链注释

// TimePartDate/TimePartTime 时间类默认值的语义分量：该关键字/函数返回值包含日期部分还是时间部分
const (
	TimePartDate = 1 << iota
	TimePartTime
)

// timeKeywordAliases 各数据库「当前日期/时间」类默认值的关键字与函数别名 → 语义分量。
// 同一语义在各源库的呈现形态互不相同：MySQL 8.0元数据为CURRENT_TIMESTAMP(3)/curdate()/curtime()，
// Oracle为SYSDATE/SYSTIMESTAMP，SQL Server为getdate()/sysdatetime()，PG与sqlite为CURRENT_*标准关键字
var timeKeywordAliases = map[string]int{
	"CURRENT_TIMESTAMP": TimePartDate | TimePartTime,
	"LOCALTIMESTAMP":    TimePartDate | TimePartTime,
	"LOCALTIME":         TimePartDate | TimePartTime,
	"NOW":               TimePartDate | TimePartTime,
	"SYSTIMESTAMP":      TimePartDate | TimePartTime,
	"GETDATE":           TimePartDate | TimePartTime,
	"SYSDATETIME":       TimePartDate | TimePartTime,
	// Oracle的SYSDATE虽名为“日期”，其DATE类型含时分秒，语义上是日期时间合一
	"SYSDATE":      TimePartDate | TimePartTime,
	"CURRENT_DATE": TimePartDate,
	"CURDATE":      TimePartDate,
	"CURRENT_TIME": TimePartTime,
	"CURTIME":      TimePartTime,
}

// ParseTimeKeywordDefault 解析日期时间类默认值原文是否为「当前日期/时间」关键字或函数。
//
// 接受无参数形态（CURRENT_TIMESTAMP、now、SYSDATE）与带一个小数秒精度参数的形态
// （CURRENT_TIMESTAMP(6)、now()），参数只允许是不超过两位的非负整数，含其他内容的函数表达式
// （如date_trunc(...)）一律不识别，避免把任意表达式当关键字裸拼。返回值为：归一大写的关键字、
// 语义分量、书写的小数秒精度（未书写参数时为-1）、是否识别
func ParseTimeKeywordDefault(val string) (keyword string, parts int, fsp int, ok bool) {
	name := strings.TrimSpace(val)
	fsp = -1
	if strings.HasSuffix(name, ")") {
		open := strings.IndexByte(name, '(')
		if open <= 0 {
			return "", 0, -1, false
		}
		inner := strings.TrimSpace(name[open+1 : len(name)-1])
		if inner != "" {
			num := 0
			if len(inner) > 2 {
				return "", 0, -1, false
			}
			for i := 0; i < len(inner); i++ {
				if inner[i] < '0' || inner[i] > '9' {
					return "", 0, -1, false
				}
				num = num*10 + int(inner[i]-'0')
			}
			fsp = num
		}
		name = strings.TrimSpace(name[:open])
	}
	upper := strings.ToUpper(name)
	parts, ok = timeKeywordAliases[upper]
	if !ok {
		return "", 0, -1, false
	}
	return upper, parts, fsp, true
}

// CanonicalTimeKeywordDefault 若默认值原文是「当前日期/时间」类关键字或函数，返回其在目标列类型下
// 应书写的SQL标准关键字形态（CURRENT_TIMESTAMP/CURRENT_DATE/CURRENT_TIME，均不带参数）。
//
// 两层归一缺一不可：
//   - 形态归一：MySQL 8.0元数据以CURRENT_TIMESTAMP(3)/curdate()/curtime()呈现，而sqlite/SQL Server等
//     写参数即语法错误，NOW()/getdate()这类源库函数名在PG等目标库也不存在；归一后不带参数与函数名，
//     目标列自身的类型与fsp决定精度，默认值语义不变
//   - 分量归一：按目标列可承载的分量取舍（如Oracle的SYSDATE迁入date列取CURRENT_DATE），
//     避免把时间戳默认值写到只接受日期的列上
//
// 非日期时间类型的列不处理：字符串列的默认值内容可能就是该文本（如varchar DEFAULT 'now'），不得被改写成关键字
func CanonicalTimeKeywordDefault(val string, dataType string) (string, bool) {
	_, parts, _, ok := ParseTimeKeywordDefault(val)
	if !ok {
		return "", false
	}
	baseType, _, _ := SplitColumnTypeBase(dataType)
	if !IsDateTimeType(baseType) {
		return "", false
	}
	switch parts & TimePartsOfBaseType(baseType) {
	case TimePartDate:
		return "CURRENT_DATE", true
	case TimePartTime:
		return "CURRENT_TIME", true
	}
	return "CURRENT_TIMESTAMP", true
}

// TimePartsOfBaseType 按目标列的基础类型名判定其可承载的语义分量：
// date类仅日期、time/timetz类仅时间、datetime/timestamp/timestamptz等日期时间合一；
// 无法识别的日期时间形态按合一处理（宁可多保留信息）
func TimePartsOfBaseType(baseType string) int {
	lower := strings.ToLower(baseType)
	switch {
	case strings.Contains(lower, "stamp"), strings.Contains(lower, "datetime"):
		return TimePartDate | TimePartTime
	case strings.Contains(lower, "date"):
		return TimePartDate
	case strings.Contains(lower, "time"):
		return TimePartTime
	}
	return TimePartDate | TimePartTime
}

// dateTimeTypeKeywords 日期时间类列的类型关键字片段（Contains匹配，datetime含time、timestamp含time）
var dateTimeTypeKeywords = []string{"date", "time"}

// IsDefaultRawExprForm 判定「字面量默认值在元数据中带引号呈现」的源库（pg/oracle/dm/mssql/sqlite）的默认值原文
// 是否为表达式而非字面量内容。
//
// 这类库的字符串字面量默认值永远带引号（SQL Server/Oracle还会额外包一层括号），因此剥去成对外层括号后
// 仍不是带引号字面量的内容，只可能是表达式：函数调用（gen_random_uuid()、pg_catalog.now()）、
// 运算式（(1 + 2)、('a' || 'b')）、裸关键字（USER）等；数字与可归一的时间关键字仍按可还原处理。
// 「当前日期/时间」类关键字（now()、SYSDATE）跨库可归一为SQL标准关键字，不视为不可还原的表达式。
// MySQL不属于本类库（字面量与表达式都呈现为去引号裸值，只能靠EXTRA区分）
func IsDefaultRawExprForm(raw string) bool {
	val := strings.TrimSpace(raw)
	for val != "" {
		if IsQuotedSqlLiteral(val) {
			return false
		}
		inner, balanced := UnwrapOuterParens(val)
		if !balanced {
			break
		}
		val = inner
	}
	if IsSqlFunctionExpr(val) {
		_, _, _, ok := ParseTimeKeywordDefault(val)
		return !ok
	}
	// 剥括号后是无参关键字/裸字面量（(CURRENT_TIMESTAMP)、((3))、NULL、TRUE）：语义与裸值一致，可原样还原
	if IsPlainSqlLiteral(val) {
		return false
	}
	// 剩下的裸文本/括号运算形态（(1 + 2)、USER、('a' || 'b')、sysdate + 1）：对字面量必带引号的源库而言
	// 不可能是默认值内容（真为内容时元数据会带引号），只能是无法跨库还原的表达式
	return true
}

// MarkExprDefault 标记列默认值是否为不可跨库还原的表达式，供字面量带引号呈现的各方言metadata读取列后调用
func MarkExprDefault(column *Column) {
	if column == nil || column.ColumnDefault == "" {
		return
	}
	column.IsExprDefault = IsDefaultRawExprForm(column.ColumnDefault)
}

// IsDateTimeType 判断列类型是否为日期时间类（date/time/datetime/timestamp/timestamptz/smalldatetime等）
func IsDateTimeType(dataType string) bool {
	return anyStringContains(strings.ToLower(dataType), dateTimeTypeKeywords)
}

// defaultStringTypeKeywords 默认值必定按字符串字面量呈现的类型关键字片段（Contains匹配）：
// 不含日期时间类（其默认值多为CURRENT_TIMESTAMP这类裸关键字，由IsPlainSqlLiteral分支处理）
var defaultStringTypeKeywords = []string{
	"char", "text", "blob", "binary", "enum", "set", "lob", "json", "string", "uuid", "var", "clob",
}

// isStringTypeColumn 判断列类型是否属于字符串/大字类（其默认值只能按字面量形态书写）
func isStringTypeColumn(dataType string) bool {
	return anyStringContains(strings.ToLower(dataType), defaultStringTypeKeywords)
}

// GenColumnDefaultSql 依据元数据返回的默认值原文生成列的 DEFAULT 子句（含前导空格），无需输出时返回空串。
//
// 各库默认值的呈现形态互不相同（MySQL 8.0为去引号的原始值、5.7/sqlite/达梦为带引号字面量、
// pg为带::cast的字面量、SQL Server为带外层括号的定义原文），而结构迁移是把**源库**的元数据喂给
// **目标方言**的生成器，故此处必须能处理所有陌生形态，旧实现在各方言重复实现且各自有误，统一按下列决策链：
//  1. 空串与裸NULL（无默认值）→ 不输出；
//  2. 纯空白原文（MySQL 8.0的 DEFAULT ' ' 呈现为单个空格）→ 按字面量引用，不可先Trim再判空而丢失；
//  3. 带引号字面量 → 还原原始值后按目标方言重新转义引用（内容可为NULL、空串、含引号/反斜杠/括号的任意文本）；
//  4. 「当前日期/时间」类关键字与函数（CURRENT_TIMESTAMP(3)、curdate()、now()、SYSDATE等，
//     含被括号整体包裹的(getdate())形态）→ 按目标列类型归一为不带参数的标准关键字
//     （CURRENT_TIMESTAMP/CURRENT_DATE/CURRENT_TIME）；
//  5. 其他函数调用形式（TO_CHAR(...)、pg_catalog.now()等无法确定语义的表达式）在非字符串列上跳过；
//     字符串列上则按字面量内容保留（MySQL 8.0去引号呈现使 varchar DEFAULT 'now()' 与裸函数调用同形，
//     而字符串列的表达式默认值必须写成 (concat(...)) 形态，裸形态几乎必然是内容）；
//  6. 可证明是数字/十六进制/位字面量/无括号关键字（如TRUE）且目标列非字符串类→ 原样裸拼；
//     字符串类列一律引用，避免与SQL关键字/数字同形的默认值内容被目标库重新解释；
//  7. 整体被括号包裹的定义原文：内层是字面量则按字面量还原；非字符串列则是表达式，跳过；
//  8. 非字符串列的不可识别裸文本：仅形如日期时间的字面量才引用，表达式（sysdate+1）跳过；
//  9. 字符串列的裸原始值（MySQL 8.0形态，如 unknown (pending)）与其余无法判定的形态按字面量引用，
//     宁可让目标库显式报类型错误，也不静默丢弃默认值；裸值形态的首尾空白属于默认值本身，不得剥除。
//
// 需要按目标库语法约束精细书写时间默认值的方言（如MySQL要求CURRENT_TIMESTAMP的小数秒与列fsp严格一致），
// 应在调用本函数前自行归一默认值原文（见mysql/sqlgen.go的mysqlTimeDefaultSql）
//
// dataType 用于区分同为 (0) 呈现的 varchar DEFAULT '(0)' 与整型表达式 DEFAULT (0)（MySQL 8.0去引号呈现导致）。
// escape 为目标方言的字面量内容转义函数（标准SQL单引号双写或mysql额外双写反斜杠）。
//
// 本入口无法获知源库信息，仅适用于默认值原文形态无歧义的场景；结构迁移/导出必须由源列元数据生成DDL，
// 应使用GenColumnDefaultSqlOf，以识别源侧标记的表达式默认值
func GenColumnDefaultSql(raw string, dataType string, escape func(string) string) string {
	return genColumnDefaultSql(raw, dataType, escape, false)
}

// GenColumnDefaultSqlOf 基于源列元数据生成DEFAULT子句：除原文形态判定外，还能识别源侧标记的
// 表达式默认值（Column.IsExprDefault）——各库函数名与表达式语法互不相通，不可安全还原时统一省略，
// 绝不退化为字符串字面量而静默污染目标表默认值
func GenColumnDefaultSqlOf(column *Column, dataType string, escape func(string) string) string {
	if column == nil {
		return ""
	}
	return genColumnDefaultSql(column.ColumnDefault, dataType, escape, column.IsExprDefault)
}

func genColumnDefaultSql(raw string, dataType string, escape func(string) string, isExprDefault bool) string {
	if raw == "" {
		return ""
	}
	// 纯空白是有意义的默认值（DEFAULT ' '），只能在保留原文的前提下判空，不能直接TrimSpace后丢弃
	val := strings.TrimSpace(raw)
	if val == "" {
		return " DEFAULT '" + escape(raw) + "'"
	}
	if val == "NULL" {
		return ""
	}
	if IsQuotedSqlLiteral(val) {
		return " DEFAULT '" + escape(UnwrapSqlLiteral(val)) + "'"
	}
	// 当前日期/时间类默认值（含MySQL 8.0元数据的CURRENT_TIMESTAMP(3)/curdate()/now()形态）必须在函数
	// 表达式判定之前处理：否则会被当作跨源不支持的函数而静默丢弃默认值
	if canonical, ok := CanonicalTimeKeywordDefault(val, dataType); ok {
		return " DEFAULT " + canonical
	}
	isStringType := isStringTypeColumn(dataType)
	// SQL Server与MySQL 8.0.13+会把表达式默认值整体括号包裹后呈现（如(getdate())），剥一层再识别
	if inner, balanced := UnwrapOuterParens(val); balanced {
		if canonical, ok := CanonicalTimeKeywordDefault(inner, dataType); ok {
			return " DEFAULT " + canonical
		}
	}
	// 源侧已标记为表达式默认值（pg的gen_random_uuid()、MySQL 8.0的(uuid())）：无法跨库还原，必须省略；
	// 旧逻辑对字符串列的裸函数形态一律按字面量保留，使DEFAULT (uuid())被写成DEFAULT 'uuid()'而静默污染默认值
	if isExprDefault || (IsSqlFunctionExpr(val) && !isStringType) {
		return ""
	}
	if IsPlainSqlLiteral(val) {
		// 二进制/位列的0x与b'01'形态是真实的SQL字面量，必须裸拼（引用后会变成同名字符而失真）
		if isBinaryOrBitType(dataType) && (isHexLiteral(val) || isBitLiteral(val)) {
			return " DEFAULT " + val
		}
		// 字符串/大字列一律引用：MySQL 8.0元数据对 DEFAULT 'TRUE'、DEFAULT 'CURRENT_TIMESTAMP'、DEFAULT '0'
		// 这类内容恰与SQL关键字/数字字面量同形的字符串值去引号呈现，裸拼会被目标库按字面量语义重新解释
		// （varchar DEFAULT TRUE 实际存为'1'属静默数据损坏，varchar DEFAULT CURRENT_TIMESTAMP 直接建表失败）
		if !isStringType {
			return " DEFAULT " + val
		}
	}
	if isParenWrapped(val) {
		inner, balanced := UnwrapOuterParens(val)
		switch {
		case !balanced && !isStringType:
			// (1)+(2) 这类非整体包裹的片段不是定义原文的外层包装，无法安全还原为字面量，省略
			return ""
		case balanced:
			if lit, ok := AsSqlStringLiteral(inner); ok {
				// ('abc')、(N'abc')：外层括号是表达式包装，内层才是字面量，还原后重新转义引用
				return " DEFAULT '" + escape(lit) + "'"
			}
			if !isStringType {
				// 非字符串列的括号形态来自表达式默认值（MySQL 8.0.13+的(1+2)、SQL Server的(getdate())）：
				// 能证明是裸字面量则还原，否则跳过，绝不把表达式写成字符串默认值
				if IsPlainSqlLiteral(inner) {
					return " DEFAULT " + inner
				}
				return ""
			}
			// 字符串列的 (0)、(pending) 形态：MySQL 8.0去引号呈现使 DEFAULT '(0)' 与表达式 (0) 不可区分，
			// 前者远比后者常见，整体按字面量引用（目标库会对其余形态报类型错误而非静默失真）
		}
	} else if !isStringType {
		// 不带括号又不是字面量/关键字的裸文本出现在非字符串列上，只可能是表达式（Oracle的sysdate+1、
		// 函数名）；除形如日期时间的字面量（必须引用）外，一律跳过，不把表达式写成字符串默认值
		if !IsDateTimeType(dataType) || !dateTimeLiteralRegexp.MatchString(val) {
			return ""
		}
	}
	// 裸值形态下原文的首尾空白属于默认值本身（MySQL 8.0去引号呈现使 DEFAULT '  x  ' 就是 空格x空格），
	// 不得TrimSpace后写回（会静默丢字符）；而括号形态的空白是定义原文的排版，取剥后内容
	quoteVal := raw
	if isParenWrapped(val) {
		quoteVal = val
	}
	return " DEFAULT '" + escape(quoteVal) + "'"
}

// dateTimeLiteralRegexp 日期时间字面量形态（2020-01-01、2020-01-01 10:00:00、2020-01-01T10:00:00.123），
// 用于判定日期时间列的裸文本默认值是字面量（必须引用）还是表达式（不得引用）
// binaryBitTypeKeywords 二进制与位类型关键字片段：其默认值的0x/b'01'形态是真实的SQL字面量，
// 不得当作文本引用（引用后'0x1234'是4个字符的文本，与二进制0x12,0x34完全不同）
var binaryBitTypeKeywords = []string{"binary", "blob", "bit", "bytea", "bytes"}

// isBinaryOrBitType 判断列类型是否属于二进制/位类型
func isBinaryOrBitType(dataType string) bool {
	return anyStringContains(strings.ToLower(dataType), binaryBitTypeKeywords)
}
