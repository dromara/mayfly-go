package dm

import (
	"mayfly-go/internal/db/dbm/dbi"
	mysql "mayfly-go/internal/db/dbm/dialect/mysql" // 触发mysql源类型注册
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCol() *dbi.Column {
	return &dbi.Column{DataType: "VARCHAR"}
}

func resolveTarget(t *testing.T, engine *dbi.TypeEngine, cat dbi.TypeCategory, col *dbi.Column) *dbi.DbDataType {
	t.Helper()
	result, err := engine.ResolveTarget(cat, col)
	require.NoError(t, err)
	return result
}

func dmTargetDialect() dbi.Dialect {
	meta := dbi.GetBackend(DbTypeDM)
	conn := &dbi.DbConn{Info: &dbi.DbInfo{Type: DbTypeDM, Backend: meta}}
	return conn.GetDialect()
}

func TestDmConverter_IntMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeDM)
	col := newCol()

	assert.Equal(t, TINYINT, resolveTarget(t, engine, dbi.TCInt1, col))
	assert.Equal(t, SMALLINT, resolveTarget(t, engine, dbi.TCInt2, col))
	assert.Equal(t, INTEGER, resolveTarget(t, engine, dbi.TCInt4, col))
	assert.Equal(t, BIGINT, resolveTarget(t, engine, dbi.TCInt8, col))
	assert.Equal(t, BIT, resolveTarget(t, engine, dbi.TCBit, col))
	// dm无无符号类型，退化为有符号；无符号小整型统一升格INT防溢出
	assert.Equal(t, INT, resolveTarget(t, engine, dbi.TCUnsignedInt1, col))
	assert.Equal(t, INT, resolveTarget(t, engine, dbi.TCUnsignedInt2, col))
	assert.Equal(t, INT, resolveTarget(t, engine, dbi.TCUnsignedInt4, col))
	assert.Equal(t, BIGINT, resolveTarget(t, engine, dbi.TCUnsignedInt8, col))
	// 整数族必须清空长度精度（clearLen副作用）
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)
}

// varchar超长边界：dm VARCHAR上限32767，超长必须转TEXT
func TestDmConverter_VarcharMaxLengthBoundary(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeDM)

	col := newCol()
	col.CharMaxLength = 32767
	assert.Equal(t, VARCHAR, resolveTarget(t, engine, dbi.TCVarchar, col))

	col2 := newCol()
	col2.CharMaxLength = 32768
	assert.Equal(t, TEXT, resolveTarget(t, engine, dbi.TCVarchar, col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

func TestDmConverter_RepresentativeMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeDM)
	col := newCol()

	assert.Equal(t, LONGVARCHAR, resolveTarget(t, engine, dbi.TCLongtext, col))
	assert.Equal(t, TEXT, resolveTarget(t, engine, dbi.TCText, col))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCBlob, col))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCLongblob, col))
	assert.Equal(t, VARCHAR, resolveTarget(t, engine, dbi.TCEnum, col))
	assert.Equal(t, VARCHAR, resolveTarget(t, engine, dbi.TCJSON, col))
	assert.Equal(t, DECIMAL, resolveTarget(t, engine, dbi.TCDecimal, col))
}

// 覆盖剩余映射（目标方言为dm时的完整映射矩阵）
func TestDmConverter_RemainingMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeDM)

	assert.Equal(t, CHAR, resolveTarget(t, engine, dbi.TCChar, newCol()))
	assert.Equal(t, NUMBER, resolveTarget(t, engine, dbi.TCNumeric, newCol()))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCBinary, newCol()))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCVarbinary, newCol()))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCMediumblob, newCol()))

	// 时间族映射及clearLength副作用（时间类型无长度语法）
	col := newCol()
	col.CharMaxLength = 100
	col.NumPrecision = 10
	assert.Equal(t, DATE, resolveTarget(t, engine, dbi.TCDate, col))
	assert.Equal(t, TIME, resolveTarget(t, engine, dbi.TCTime, col))
	assert.Equal(t, DATETIME, resolveTarget(t, engine, dbi.TCDateTime, col))
	assert.Equal(t, TIMESTAMP, resolveTarget(t, engine, dbi.TCTimestamp, col))
	assert.Equal(t, 0, col.CharMaxLength)
	assert.Equal(t, 0, col.NumPrecision)

	// mediumtext归一化为text并清空长度（text无长度语法）
	col2 := newCol()
	col2.CharMaxLength = 500
	assert.Equal(t, TEXT, resolveTarget(t, engine, dbi.TCMediumtext, col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

// 端到端回归：mysql 超长varchar 迁移至 dm 的完整链路（历史上曾生成非法DDL VARCHAR(50000)）
func TestDmConvToTargetDbColumn_FromMysql(t *testing.T) {
	dbi.GetBackend(mysql.DbTypeMysql)
	dialect := dmTargetDialect()

	longVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 50000}
	err := dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeDM, dialect, longVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "TEXT", longVarchar.DataType)
	assert.Equal(t, "TEXT", longVarchar.GetColumnType())

	// 正常长度保留
	normalVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 50}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeDM, dialect, normalVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "VARCHAR(50)", normalVarchar.GetColumnType())
}
