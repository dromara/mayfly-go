package application

import (
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/importer"
)

// TestDropAllEmptyAutoIncrement 自增列整列为空时剔除、有显式值时保留、非自增列永不剔除。
func TestDropAllEmptyAutoIncrement(t *testing.T) {
	cols := []dbi.Column{{ColumnName: "id", AutoIncrement: true}, {ColumnName: "name"}}
	srcs := []int{0, 1}

	// id 全空 → 剔除，仅留 name
	gotCols, gotSrc := dropAllEmptyAutoIncrement(&importer.Table{Rows: [][]string{{"", "a"}, {"", "b"}}}, cols, srcs)
	if len(gotCols) != 1 || gotCols[0].ColumnName != "name" || len(gotSrc) != 1 || gotSrc[0] != 1 {
		t.Fatalf("全空自增列应被剔除: cols=%v src=%v", gotCols, gotSrc)
	}

	// id 有非空值 → 保留两列（用户显式指定主键）
	gotCols2, _ := dropAllEmptyAutoIncrement(&importer.Table{Rows: [][]string{{"5", "a"}, {"", "b"}}}, cols, srcs)
	if len(gotCols2) != 2 {
		t.Fatalf("自增列有显式值时应保留, got %d", len(gotCols2))
	}

	// 非自增列即使全空也不剔除（那是用户要写空值的普通列）
	cols3 := []dbi.Column{{ColumnName: "note"}, {ColumnName: "name"}}
	gotCols3, _ := dropAllEmptyAutoIncrement(&importer.Table{Rows: [][]string{{"", "a"}}}, cols3, []int{0, 1})
	if len(gotCols3) != 2 {
		t.Fatalf("非自增列不应被剔除, got %d", len(gotCols3))
	}
}
