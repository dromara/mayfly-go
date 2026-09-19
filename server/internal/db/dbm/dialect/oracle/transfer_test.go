package oracle

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

func oracleTargetDialect() dbi.Dialect {
	meta := dbi.GetBackend(DbTypeOracle)
	conn := &dbi.DbConn{Info: &dbi.DbInfo{Type: DbTypeOracle, Backend: meta}}
	return conn.GetDialect()
}

func TestOracleConverter_IntMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeOracle)
	col := newCol()

	assert.Equal(t, TINYINT, resolveTarget(t, engine, dbi.TCInt1, col))
	assert.Equal(t, SMALLINT, resolveTarget(t, engine, dbi.TCInt2, col))
	assert.Equal(t, INTEGER, resolveTarget(t, engine, dbi.TCInt4, col))
	assert.Equal(t, BIGINT, resolveTarget(t, engine, dbi.TCInt8, col))
	assert.Equal(t, BIT, resolveTarget(t, engine, dbi.TCBit, col))
	// oracle无无符号类型，退化为有符号
	assert.Equal(t, INT, resolveTarget(t, engine, dbi.TCUnsignedInt1, col))
	assert.Equal(t, BIGINT, resolveTarget(t, engine, dbi.TCUnsignedInt8, col))
}

// varchar超长边界：VARCHAR2(n)上限4000字节（extended默认关闭），超长必须转CLOB
func TestOracleConverter_VarcharMaxLengthBoundary(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeOracle)

	col := newCol()
	col.CharMaxLength = 4000
	assert.Equal(t, VARCHAR2, resolveTarget(t, engine, dbi.TCVarchar, col))

	col2 := newCol()
	col2.CharMaxLength = 4001
	assert.Equal(t, CLOB, resolveTarget(t, engine, dbi.TCVarchar, col2))
	assert.Equal(t, 0, col2.CharMaxLength)
}

// 长文本映射CLOB：NVARCHAR2无长度时默认长度为1，长文本必插入失败（回归守护）
func TestOracleConverter_TextFamilyToClob(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeOracle)

	for _, cat := range []dbi.TypeCategory{dbi.TCText, dbi.TCMediumtext, dbi.TCLongtext} {
		col := newCol()
		col.CharMaxLength = 12345
		assert.Equal(t, CLOB, resolveTarget(t, engine, cat, col))
		// CLOB无长度概念，必须清空源长度
		assert.Equal(t, 0, col.CharMaxLength)
	}
}

func TestOracleConverter_RepresentativeMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeOracle)
	col := newCol()

	assert.Equal(t, DATE, resolveTarget(t, engine, dbi.TCDate, col))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCBlob, col))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCLongblob, col))
	assert.Equal(t, NUMBER, resolveTarget(t, engine, dbi.TCNumeric, col))
	assert.Equal(t, NVARCHAR2, resolveTarget(t, engine, dbi.TCEnum, col))
}

// 覆盖剩余映射（目标方言为oracle时的完整映射矩阵）
func TestOracleConverter_RemainingMappings(t *testing.T) {
	engine := dbi.GetTypeEngine(DbTypeOracle)

	assert.Equal(t, CHAR, resolveTarget(t, engine, dbi.TCChar, newCol()))
	// 无精度约束的精确数值（pg的numeric、oracle自己的NUMBER）必须落为不带参数的NUMBER：
	// Oracle语义下无参DECIMAL即NUMBER(38,0)，沿用会静默截断所有小数
	assert.Equal(t, NUMBER, resolveTarget(t, engine, dbi.TCDecimal, newCol()))
	// 带精度时落DECIMAL(p,s)，并将超出Oracle上限38位的精度收敛（mysql decimal(65,30)不收敛即非法DDL）
	decCol := newCol()
	decCol.NumPrecision, decCol.NumScale = 65, 30
	assert.Equal(t, DECIMAL, resolveTarget(t, engine, dbi.TCDecimal, decCol))
	assert.Equal(t, 38, decCol.NumPrecision)
	assert.Equal(t, 30, decCol.NumScale)
	// oracle无无符号类型，退化为容量足够的INT
	assert.Equal(t, INT, resolveTarget(t, engine, dbi.TCUnsignedInt1, newCol()))
	assert.Equal(t, INT, resolveTarget(t, engine, dbi.TCUnsignedInt2, newCol()))
	assert.Equal(t, INT, resolveTarget(t, engine, dbi.TCUnsignedInt4, newCol()))
	assert.Equal(t, TIME, resolveTarget(t, engine, dbi.TCTime, newCol()))
	// datetime/timestamp统一归TIMESTAMP
	assert.Equal(t, TIMESTAMP, resolveTarget(t, engine, dbi.TCDateTime, newCol()))
	assert.Equal(t, TIMESTAMP, resolveTarget(t, engine, dbi.TCTimestamp, newCol()))
	// 二进制族统一BLOB
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCBinary, newCol()))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCVarbinary, newCol()))
	assert.Equal(t, BLOB, resolveTarget(t, engine, dbi.TCMediumblob, newCol()))
	// oracle无原生JSON，用NVARCHAR2承载
	assert.Equal(t, NVARCHAR2, resolveTarget(t, engine, dbi.TCJSON, newCol()))
}

// 端到端回归：mysql text/超长varchar 迁移至 oracle 的完整链路
func TestOracleConvToTargetDbColumn_FromMysql(t *testing.T) {
	dbi.GetBackend(mysql.DbTypeMysql)
	dialect := oracleTargetDialect()

	// longtext → CLOB，历史上曾映射NVARCHAR2（默认长度1导致长文本插入失败）
	textCol := &dbi.Column{DataType: "longtext"}
	err := dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeOracle, dialect, textCol)
	assert.NoError(t, err)
	assert.Equal(t, "CLOB", textCol.DataType)
	assert.Equal(t, "CLOB", textCol.GetColumnType())

	// varchar(5000) → CLOB，历史上曾生成非法DDL VARCHAR2(5000)
	longVarchar := &dbi.Column{DataType: "varchar", CharMaxLength: 5000}
	err = dbi.ConvToTargetDbColumn(mysql.DbTypeMysql, DbTypeOracle, dialect, longVarchar)
	assert.NoError(t, err)
	assert.Equal(t, "CLOB", longVarchar.DataType)
	assert.Equal(t, "CLOB", longVarchar.GetColumnType())
}
