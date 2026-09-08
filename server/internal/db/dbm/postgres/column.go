package postgres

import (
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
)

// pgSQLValueBytes 二进制值转SQL：hex编码输出postgres bytea标准hex格式'\x...'保真还原，
// 非hex值保留特殊字符的字符串字面量处理
func pgSQLValueBytes(val any) string {
	if val == nil {
		return dbi.NULL
	}
	if strVal, ok := val.(string); ok && dbi.IsHexString(strVal) {
		return fmt.Sprintf("'\\x%s'", strVal)
	}
	return dbi.SQLValuePreserveSpecialChars(val)
}

// pgSQLValueBit pg的bit列值转SQL：lib/pq对bit/bit varying列读回文本位串（如"11111111"），
// 输出'位串'字符串字面量——pg对unknown literal到bit列自动做assignment cast，无损还原；
// 不能输出裸数字：pg不允许integer到bit的隐式assignment（如255会报错）；
// 非位串形态（脏数据）退化为转义字符串字面量交由目标库校验
func pgSQLValueBit(val any) string {
	if val == nil {
		return dbi.NULL
	}
	s := fmt.Sprintf("%v", val)
	if isBitText(s) {
		return fmt.Sprintf("'%s'", s)
	}
	return dbi.SQLValueString(val)
}

func isBitText(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != '0' && s[i] != '1' {
			return false
		}
	}
	return true
}

var (
	// DTBytesPg postgres专用二进制类型：hex值以bytea标准hex格式保真还原
	DTBytesPg = dbi.DTBytes.Copy().WithSQLValue(pgSQLValueBytes)

	// DTBitPg postgres专用bit类型：bit列读回为文本位串（与mysql的位字节形态完全不同，
	// 不能复用DTBit——bitValuer按大端位字节合成int64，文本'1'的字节0x31会被算出错值），
	// 故valuer用string保留位串文本，SQLValue输出'位串'字面量
	DTBitPg = dbi.DTString.Copy().WithSQLValue(pgSQLValueBit)

	Bool = dbi.NewDbDataType("bool", dbi.DTString).WithCT(dbi.CTBool).WithFixColumn(dbi.ClearNumPrecision)
	// pg整型不接受任何类型修饰符，而information_schema会为其回报numeric_precision（int2=16、int4=32、int8=64），
	// 必须清精度：仅ClearNumScale会使GetColumnType拼出int4(32)这类非法DDL（pg报 type modifier is not allowed）
	Int2 = dbi.NewDbDataType("int2", dbi.DTInt16).WithCT(dbi.CTInt2).WithFixColumn(dbi.ClearNumPrecision)
	Int4 = dbi.NewDbDataType("int4", dbi.DTInt32).WithCT(dbi.CTInt4).WithFixColumn(dbi.ClearNumPrecision)
	Int8 = dbi.NewDbDataType("int8", dbi.DTInt64).WithCT(dbi.CTInt8).WithFixColumn(dbi.ClearNumPrecision)
	// pg的numeric是任意精度精确数值（与decimal完全同义），必须归CTDecimal：若归CTNumeric，
	// 目标为MySQL时会被映射为double而失去小数精度（如numeric(20,6)转double后仅保留~15位有效数字）
	Numeric = dbi.NewDbDataType("numeric", dbi.DTNumeric).WithCT(dbi.CTDecimal)
	Decimal = dbi.NewDbDataType("decimal", dbi.DTDecimal).WithCT(dbi.CTDecimal)

	// pg元数据的data_type取pg_type.typname（内部规范名），float4/float8/bpchar等为SQL写法之外的形式，
	// 未注册时会回退为DefaultDbDataType(CTVarchar)，使浮点/字符列迁移到强类型库时被静默改成varchar
	// pg的float4/float8同样不接受类型修饰符，但元数据回报numeric_precision（24/53位有效位），
	// 不清会生成float4(24)这类非法DDL；同money一样属于无参数类型
	Float4      = dbi.NewDbDataType("float4", dbi.DTNumeric).WithCT(dbi.CTNumeric).WithFixColumn(dbi.ClearNumPrecision)
	Float8      = dbi.NewDbDataType("float8", dbi.DTNumeric).WithCT(dbi.CTNumeric).WithFixColumn(dbi.ClearNumPrecision)
	Bpchar      = dbi.NewDbDataType("bpchar", dbi.DTStringPreserveSpecial).WithCT(dbi.CTChar)
	Smallserial = dbi.NewDbDataType("smallserial", dbi.DTInt16).WithCT(dbi.CTInt2).WithFixColumn(dbi.ClearNumPrecision)
	Serial      = dbi.NewDbDataType("serial", dbi.DTInt32).WithCT(dbi.CTInt4).WithFixColumn(dbi.ClearNumPrecision)
	Bigserial   = dbi.NewDbDataType("bigserial", dbi.DTInt64).WithCT(dbi.CTInt8).WithFixColumn(dbi.ClearNumPrecision)
	Largeserial = dbi.NewDbDataType("largeserial", dbi.DTInt64).WithCT(dbi.CTInt8).WithFixColumn(dbi.ClearNumPrecision)

	Money = dbi.NewDbDataType("money", dbi.DTString).WithCT(dbi.CTVarchar).WithFixColumn(dbi.ClearNumPrecision)

	Char    = dbi.NewDbDataType("char", dbi.DTStringPreserveSpecial).WithCT(dbi.CTChar)
	Nchar   = dbi.NewDbDataType("nchar", dbi.DTStringPreserveSpecial).WithCT(dbi.CTVarchar)
	Varchar = dbi.NewDbDataType("varchar", dbi.DTStringPreserveSpecial).WithCT(dbi.CTVarchar)
	Text    = dbi.NewDbDataType("text", dbi.DTStringPreserveSpecial).WithCT(dbi.CTText).WithFixColumn(dbi.ClearCharMaxLength)
	Json    = dbi.NewDbDataType("json", dbi.DTStringPreserveSpecial).WithCT(dbi.CTJSON).WithFixColumn(dbi.ClearCharMaxLength)
	Jsonb   = dbi.NewDbDataType("jsonb", dbi.DTStringPreserveSpecial).WithCT(dbi.CTJSON).WithFixColumn(dbi.ClearCharMaxLength)
	Bytea   = dbi.NewDbDataType("bytea", DTBytesPg).WithCT(dbi.CTBinary)

	// pg的bit(默认1bit)与bit varying(内部名varbit，information_schema呈现为"bit varying")，
	// 三个变体名均需注册（pg_type.typname为varbit，元数据data_type可能呈现为bit varying）
	Bit        = dbi.NewDbDataType("bit", DTBitPg).WithCT(dbi.CTBit)
	Varbit     = dbi.NewDbDataType("varbit", DTBitPg).WithCT(dbi.CTBit)
	BitVarying = dbi.NewDbDataType("bit varying", DTBitPg).WithCT(dbi.CTBit)

	// 时间类列只需抹掉（恒为空的）字符长度，不能抹精度：pg的timestamp/time小数秒精度存于NumPrecision，
	// 清掉会使异构迁移目标库按自身默认精度建表而静默丢失小数秒
	Date      = dbi.NewDbDataType("date", dbi.DTDate).WithCT(dbi.CTDate).WithFixColumn(dbi.ClearCharLength)
	Time      = dbi.NewDbDataType("time", dbi.DTTime).WithCT(dbi.CTTime).WithFixColumn(dbi.ClearCharLength)
	Timetz    = dbi.NewDbDataType("timetz", dbi.DTTime).WithCT(dbi.CTTime).WithFixColumn(dbi.ClearCharLength)
	Timestamp = dbi.NewDbDataType("timestamp", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(dbi.ClearCharLength)

	// 带时区类型与无时区类型共用同一公共类型：目标库无时区感知类型时至少保证落为日期时间语义而非varchar
	Timestamptz = dbi.NewDbDataType("timestamptz", dbi.DTDateTime).WithCT(dbi.CTDateTime).WithFixColumn(dbi.ClearCharLength)
)
