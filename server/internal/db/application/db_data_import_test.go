package application

import (
	"testing"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

// TestDropAllEmptyAutoIncrement 自增列整列为空时剔除、有显式值时保留、非自增列永不剔除。
// anyNonEmpty[i] 表示文件第 i 列是否出现过非空单元格（第一遍流式扫描结果）。
func TestDropAllEmptyAutoIncrement(t *testing.T) {
	cols := []dbi.Column{{ColumnName: "id", AutoIncrement: true}, {ColumnName: "name"}}
	srcs := []int{0, 1}

	// id 全空 → 剔除，仅留 name
	gotCols, gotSrc := dropAllEmptyAutoIncrement([]bool{false, true}, cols, srcs)
	if len(gotCols) != 1 || gotCols[0].ColumnName != "name" || len(gotSrc) != 1 || gotSrc[0] != 1 {
		t.Fatalf("全空自增列应被剔除: cols=%v src=%v", gotCols, gotSrc)
	}

	// id 有非空值 → 保留两列（用户显式指定主键）
	gotCols2, _ := dropAllEmptyAutoIncrement([]bool{true, true}, cols, srcs)
	if len(gotCols2) != 2 {
		t.Fatalf("自增列有显式值时应保留, got %d", len(gotCols2))
	}

	// 非自增列即使全空也不剔除（那是用户要写空值的普通列）
	cols3 := []dbi.Column{{ColumnName: "note"}, {ColumnName: "name"}}
	gotCols3, _ := dropAllEmptyAutoIncrement([]bool{false, false}, cols3, []int{0, 1})
	if len(gotCols3) != 2 {
		t.Fatalf("非自增列不应被剔除, got %d", len(gotCols3))
	}
}

// TestClampImportBatchSize batchSize 来自请求参数，必须收敛到固定上限：
// 否则 make([][]any, 0, batchSize) 会在读数据前按用户值预分配，1e9 直接不可 recover 地 OOM。
func TestClampImportBatchSize(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, dto.DefaultImportBatchSize},
		{-1, dto.DefaultImportBatchSize},
		{1, 1},
		{maxImportBatchSize, maxImportBatchSize},
		{maxImportBatchSize + 1, maxImportBatchSize},
		{1_000_000_000, maxImportBatchSize},
	}
	for _, c := range cases {
		if got := clampImportBatchSize(c.in); got != c.want {
			t.Fatalf("clampImportBatchSize(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
