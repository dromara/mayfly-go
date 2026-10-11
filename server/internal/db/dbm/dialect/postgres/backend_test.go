package postgres

import (
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// libpq的keyword/value格式：值含空白时必须单引号包裹，值中的单引号与反斜杠需转义。
// 原实现未转义，密码含空格/特殊字符时会导致DSN解析错误或连接参数被截断
func TestEscapeLibpqParam(t *testing.T) {
	// 普通值原样返回（不加引号）
	assert.Equal(t, "localhost", escapeLibpqParam("localhost"))
	assert.Equal(t, "my_db", escapeLibpqParam("my_db"))

	// 含空格需包裹
	assert.Equal(t, "'my pass word'", escapeLibpqParam("my pass word"))

	// 含tab/换行同样包裹
	assert.Equal(t, "'a\tb'", escapeLibpqParam("a\tb"))
	assert.Equal(t, "'a\nb'", escapeLibpqParam("a\nb"))

	// 单引号转义（值含空白判定命中因含单引号，需包裹）
	assert.Equal(t, `'it\'s'`, escapeLibpqParam(`it's`))

	// 反斜杠转义
	assert.Equal(t, `'a\\b'`, escapeLibpqParam(`a\b`))

	// 单引号与反斜杠混合：先转反斜杠再转单引号
	assert.Equal(t, `'p\\\'w'`, escapeLibpqParam(`p\'w`))

	// 空串原样返回
	assert.Equal(t, "", escapeLibpqParam(""))
}

func newCol() *dbi.Column {
	return &dbi.Column{DataType: "varchar"}
}

func resolveTarget(t *testing.T, engine *dbi.TypeEngine, cat dbi.TypeCategory, col *dbi.Column) *dbi.DbDataType {
	t.Helper()
	result, err := engine.ResolveTarget(cat, col)
	require.NoError(t, err)
	return result
}

func TestPostgresConverter_IntMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypePostgres)
	col := newCol()

	assert.Equal(t, Int2, resolveTarget(t, engine, dbi.TCInt1, col))
	assert.Equal(t, Int2, resolveTarget(t, engine, dbi.TCInt2, col))
	assert.Equal(t, Int4, resolveTarget(t, engine, dbi.TCInt4, col))
	assert.Equal(t, Int8, resolveTarget(t, engine, dbi.TCInt8, col))
	// pg无无符号类型，退化为有符号
	assert.Equal(t, Int2, resolveTarget(t, engine, dbi.TCUnsignedInt1, col))
	// 无符号需升位避免溢出：uint16>int2上限、uint32>int4上限、uint64>int8上限
	assert.Equal(t, Int4, resolveTarget(t, engine, dbi.TCUnsignedInt2, col))
	assert.Equal(t, Int8, resolveTarget(t, engine, dbi.TCUnsignedInt4, col))
	assert.Equal(t, Numeric, resolveTarget(t, engine, dbi.TCUnsignedInt8, col))
	// 整数族清空长度精度（cleanupColumnParams副作用）
	assert.Equal(t, 0, col.NumPrecision)
	assert.Equal(t, 0, col.CharMaxLength)
}

func TestPostgresConverter_RepresentativeMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypePostgres)
	col := newCol()

	// 字符串族：pg用标准DTString（standard_conforming_strings=on下反斜杠不转义）
	assert.Equal(t, Varchar, resolveTarget(t, engine, dbi.TCVarchar, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCMediumtext, col))
	assert.Equal(t, Text, resolveTarget(t, engine, dbi.TCLongtext, col))
	// 二进制族归一到bytea
	assert.Equal(t, Bytea, resolveTarget(t, engine, dbi.TCBlob, col))
	assert.Equal(t, Bytea, resolveTarget(t, engine, dbi.TCLongblob, col))
	// 时间族
	assert.Equal(t, Timestamp, resolveTarget(t, engine, dbi.TCDateTime, col))
	assert.Equal(t, Timestamp, resolveTarget(t, engine, dbi.TCTimestamp, col))
	// json
	assert.Equal(t, Json, resolveTarget(t, engine, dbi.TCJSON, col))
}

// TestPostgresNativeExtensionTypesRecognised pg 的 uuid 与数组列必须被类型引擎认出来。
//
// 存在理由：两者曾因未注册而在源类型解析处直接失败——异构迁移整表报「data type not found」，
// 结构比对两侧都「认不出」而只能标比不了。数组按 `_` 前缀解析而非逐元素类型登记，
// 因此元素类型认不出（自定义枚举/域）时数组仍算认不出，不得猜类别。
func TestPostgresNativeExtensionTypesRecognised(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypePostgres)

	uuidType, ok := engine.ResolveType("uuid")
	require.True(t, ok, "uuid 列必须可解析")
	assert.Equal(t, dbi.TCUUID, uuidType.Category())

	array, ok := engine.ResolveType("_int8")
	require.True(t, ok, "bigint[] 的元数据名 _int8 必须可解析")
	assert.Equal(t, dbi.TCArray, array.Category())
	assert.Equal(t, "_int8", array.Name, "类型名须保持库侧形态，不得暴露注册用的占位名")
	_, ok = engine.ResolveType("_myenum")
	assert.False(t, ok, "元素类型未注册的数组不得猜类别")

	// 目标为 pg 时 uuid 保持原生类型（退化成 varchar 会让主键/索引列变成宽字符串）
	assert.Equal(t, Uuid, resolveTarget(t, engine, dbi.TCUUID, newCol()))
	// 字符串列不会被误判成 uuid：TCUUID 规则只对源类别为 uuid 的列生效
	assert.Equal(t, Varchar, resolveTarget(t, engine, dbi.TCVarchar, newCol()))

	// uuid 不接受类型修饰符：元数据若带长度必须在读侧抹掉，否则 DDL 拼成 uuid(36) 直接建表失败
	col := &dbi.Column{DataType: "uuid", CharMaxLength: 36}
	uuidType.FixColumn(col)
	assert.Equal(t, 0, col.CharMaxLength)
}
