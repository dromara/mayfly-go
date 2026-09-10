package mssql

// mssql 列元数据形态归一与「不限长类型异构迁出」的无环境依赖单测。
//
// 钉死两类只在真实迁移中暴露、但逻辑本身纯函数的缺陷：
//  1. sys.columns 对 varchar(max)/nvarchar(max)/varbinary(max) 回报 max_length = -1，
//     而 nchar/nvarchar 还需把字节数换算成字符数；Go 整型除法向零截断使 -1/2 == 0，
//     两者顺序颠倒会让 nvarchar(max) 退化为“长度0的 nvarchar”——同方言DDL拼成 nvarchar(1)
//     （所有超长值静默截断且无报错），异构迁移因取不到长度被目标方言兜底为 varchar(255)。
//  2. 无限长类型必须归文本/大对象公共类型（CTLongtext/CTLongblob）迁出，与 oracle/dm 对
//     CLOB/TEXT 的处理同构；归 CTVarchar 会使异构目标按定长列建表。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"

	_ "mayfly-go/internal/db/dbm/mysql"    // 触发目标方言类型注册
	_ "mayfly-go/internal/db/dbm/postgres" // 触发目标方言类型注册
	_ "mayfly-go/internal/db/dbm/sqlite"   // 触发目标方言类型注册
)

func TestNormalizeMssqlColumnType(t *testing.T) {
	// (max)形态：-1必须还原进类型名，且绝不能被字节→字符换算吞掉
	for _, dt := range []string{"varchar", "nvarchar", "varbinary"} {
		name, length := normalizeMssqlColumnType(dt, -1)
		assert.Equal(t, dt+"(max)", name, "[%s] -1未还原为(max)形态", dt)
		assert.Equal(t, 0, length, "[%s] (max)列不应残留长度", dt)
	}

	// nchar/nvarchar按字节回报，DDL需字符数
	name, length := normalizeMssqlColumnType("nvarchar", 4000)
	assert.Equal(t, "nvarchar", name)
	assert.Equal(t, 2000, length, "nvarchar的字节长度必须换算为字符数")
	_, length = normalizeMssqlColumnType("nchar", 2)
	assert.Equal(t, 1, length)

	// varchar/varbinary本身按字节计，不换算；text等LOB列的16由自身FixColumn清空，此处不动
	name, length = normalizeMssqlColumnType("varchar", 8000)
	assert.Equal(t, "varchar", name)
	assert.Equal(t, 8000, length)
	name, length = normalizeMssqlColumnType("text", 16)
	assert.Equal(t, "text", name)
	assert.Equal(t, 16, length)
}

// targetDialectOf 构造目标方言的伪连接（不连库），供结构转换单测使用
func targetDialectOf(dt dbi.DbType) dbi.Dialect {
	conn := &dbi.DbConn{Info: &dbi.DbInfo{Type: dt, Meta: dbi.GetMeta(dt)}}
	return conn.GetDialect()
}

// TestUnboundedTypesMigrateAsText 不限长类型异构迁出必须是目标方言的大文本/大对象类型
func TestUnboundedTypesMigrateAsText(t *testing.T) {
	// 公共类型转换器由首次GetMeta时懒注册，本用例以mssql为源，必须显式触发其注册
	require.NotNil(t, dbi.GetMeta(DbTypeMssql))
	cases := []struct {
		src      string
		target   dbi.DbType
		wantType string
	}{
		// 归CTVarchar时mysql会因取不到长度兜底为varchar(255)（超长值报Data too long），本断言即变红
		{"nvarchar(max)", "mysql", "longtext"},
		{"varchar(max)", "mysql", "longtext"},
		{"text", "mysql", "longtext"},
		{"ntext", "mysql", "longtext"},
		{"varbinary(max)", "mysql", "longblob"},
		{"nvarchar(max)", "postgres", "text"},
		{"varbinary(max)", "postgres", "bytea"},
		{"nvarchar(max)", "sqlite", "text"},
	}
	for _, cc := range cases {
		cc := cc
		t.Run(cc.src+"->"+string(cc.target), func(t *testing.T) {
			col := &dbi.Column{ColumnName: "c", DataType: cc.src}
			require.NoError(t, dbi.ConvToTargetDbColumn(DbTypeMssql, cc.target,
				targetDialectOf(cc.target), col))
			assert.Equal(t, cc.wantType, col.GetColumnType(),
				"不限长源列被迁成定长/错误类型，超长数据会在目标库静默截断或报错")
		})
	}

	// 有界列仍须保留长度语义（防止矫枉过正：全部落longtext会让目标库丢失varchar约束）
	col := &dbi.Column{ColumnName: "c", DataType: "nvarchar", CharMaxLength: 50}
	require.NoError(t, dbi.ConvToTargetDbColumn(DbTypeMssql, "mysql", targetDialectOf("mysql"), col))
	assert.Equal(t, "varchar(50)", col.GetColumnType())
}
