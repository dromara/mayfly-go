package application

import (
	"testing"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/stretchr/testify/require"
)

// TestBuildImportColumnRefsCoversEveryMappedColumn 导入的列级 ACL 必须覆盖每个映射列。
//
// 历史缺陷：这里曾直接传 nil 给 AssertStatementAllowed，而 AssertColumnsAllowed 对空 refs
// 直接放行 —— 表级判定绿、列级保护整条被绕过（可以覆盖敏感列取值）。
func TestBuildImportColumnRefsCoversEveryMappedColumn(t *testing.T) {
	cols := []dbi.Column{
		{ColumnName: "id"},
		{ColumnName: "phone"},
		{ColumnName: "salary"},
	}
	refs := buildImportColumnRefs("t_emp", cols)

	require.Len(t, refs, len(cols), "每个映射列都必须进入判定集合，不能漏")
	for i, ref := range refs {
		require.Equal(t, "t_emp", ref.Table)
		require.Equal(t, cols[i].ColumnName, ref.Column)
	}

	// 空列名不构造引用（避免产生一条无意义的 table."" 判定）
	require.Len(t, buildImportColumnRefs("t", []dbi.Column{{ColumnName: ""}}), 0)
	// 无映射列时返回空集合（此时上层不会调用判定；这是显式语义，不是"随便传 nil"）
	require.Empty(t, buildImportColumnRefs("t", nil))
}
