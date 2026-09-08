package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDataTypes 声明类型的类型名/长度/精度切割：SQLite按DDL原文保存声明类型，
// 必须容忍常见的书写空白（decimal(10, 2)、varchar (100)），否则整串被当作类型名而落入
// 未注册类型的varchar兼容，结构迁移时数值与长度语义静默失真
func TestGetDataTypes(t *testing.T) {
	sd := &SqliteMetadata{}
	kases := []struct {
		declaredType string
		dataType     string
		length       string
		scale        string
	}{
		{"decimal(10,2)", "decimal", "10", "2"},
		{"DECIMAL(20,6)", "DECIMAL", "20", "6"},
		{"decimal(10, 2)", "decimal", "10", "2"},
		{"decimal (10 , 2)", "decimal", "10", "2"},
		{"numeric(10,0)", "numeric", "10", "0"},
		{"varchar(100)", "varchar", "100", ""},
		{"varchar (100)", "varchar", "100", ""},
		{"VARCHAR( 255 )", "VARCHAR", "255", ""},
		{"int(11) unsigned", "int", "11", ""},
		{"text", "text", "", ""},
		{"timestamp", "timestamp", "", ""},
		{"double precision", "double precision", "", ""},
		{"unsigned big int", "unsigned big int", "", ""},
		{"", "", "", ""},
	}

	for _, k := range kases {
		dataType, length, scale := sd.getDataTypes(k.declaredType)
		assert.Equal(t, k.dataType, dataType, "declaredType=%q", k.declaredType)
		assert.Equal(t, k.length, length, "declaredType=%q", k.declaredType)
		assert.Equal(t, k.scale, scale, "declaredType=%q", k.declaredType)
	}
}

// TestGetDataTypes_RealSqliteDeclaredTypes 用真实SQLite验证前提：库内报告的声明类型
// 与建表DDL原文一致（保留大小写与书写空白），因此切割必须能处理这些真实形态
func TestGetDataTypes_RealSqliteDeclaredTypes(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	declaredTypes := []string{"decimal(10,2)", "decimal(10, 2)", "varchar (100)", "double precision", "timestamp"}
	var sb strings.Builder
	sb.WriteString("CREATE TABLE t_decl (id INTEGER PRIMARY KEY")
	for i, dt := range declaredTypes {
		sb.WriteString(", c" + string(rune('a'+i)) + " " + dt)
	}
	sb.WriteString(")")
	_, err = db.Exec(sb.String())
	require.NoError(t, err)

	rows, err := db.Query("PRAGMA table_info(t_decl)")
	require.NoError(t, err)
	defer rows.Close()

	declared := make([]string, 0, len(declaredTypes))
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull int
		var dflt any
		var pk int
		require.NoError(t, rows.Scan(&cid, &name, &dtype, &notnull, &dflt, &pk))
		if name != "id" {
			declared = append(declared, dtype)
		}
	}
	require.NoError(t, rows.Err())
	require.Len(t, declared, len(declaredTypes))

	sd := &SqliteMetadata{}
	// 库内报告的声明类型必须与DDL原文一致（SQLite不做类型归一化）
	assert.Equal(t, declaredTypes, declared)
	// 真实声明形态必须能切出类型名与参数（double precision/timestamp本无参数）
	wantTypes := []string{"decimal", "decimal", "varchar", "double precision", "timestamp"}
	wantLengths := []string{"10", "10", "100", "", ""}
	wantScales := []string{"2", "2", "", "", ""}
	for i, dtype := range declared {
		gotType, gotLen, gotScale := sd.getDataTypes(dtype)
		assert.Equal(t, wantTypes[i], gotType, "declared=%q", dtype)
		assert.Equal(t, wantLengths[i], gotLen, "declared=%q", dtype)
		assert.Equal(t, wantScales[i], gotScale, "declared=%q", dtype)
	}
}
