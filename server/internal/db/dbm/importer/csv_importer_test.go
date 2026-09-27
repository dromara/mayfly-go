package importer

import (
	"strings"
	"testing"
)

// TestCSVImporterParse 锁定 CSV 解析的关键行为：表头识别、分隔符、不等宽行、预览行数限制。
//
// 这些点若静默出错，导入会表现为「成功但数据错位/丢列」，故用真实文本断言读回内容。
func TestCSVImporterParse(t *testing.T) {
	imp := &CSVImporter{}

	t.Run("含表头_逗号分隔", func(t *testing.T) {
		table, err := imp.Parse(strings.NewReader("id,name\n1,alice\n2,bob\n"), &Options{HasHeader: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !table.HasHeader || strings.Join(table.Headers, ",") != "id,name" {
			t.Fatalf("headers wrong: %+v", table)
		}
		if len(table.Rows) != 2 || table.Rows[1][1] != "bob" {
			t.Fatalf("rows wrong: %+v", table.Rows)
		}
	})

	t.Run("无表头_首行即数据", func(t *testing.T) {
		table, err := imp.Parse(strings.NewReader("1,alice\n2,bob\n"), &Options{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if table.HasHeader || len(table.Headers) != 0 {
			t.Fatalf("should have no header: %+v", table)
		}
		if len(table.Rows) != 2 || table.Rows[0][0] != "1" {
			t.Fatalf("rows wrong: %+v", table.Rows)
		}
	})

	t.Run("分号分隔", func(t *testing.T) {
		table, err := imp.Parse(strings.NewReader("a;b;c\n1;2;3\n"), &Options{HasHeader: true, FieldSeparator: ";"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Join(table.Headers, "|") != "a|b|c" {
			t.Fatalf("semicolon separator not applied: %+v", table.Headers)
		}
	})

	t.Run("不等宽行不因字段数报错", func(t *testing.T) {
		// 尾列缺失：CSV 常见，交由上层按列映射对齐，此处必须成功解析
		table, err := imp.Parse(strings.NewReader("id,name,age\n1,alice\n"), &Options{HasHeader: true})
		if err != nil {
			t.Fatalf("ragged rows must not error: %v", err)
		}
		if len(table.Rows[0]) != 2 {
			t.Fatalf("ragged row should keep its 2 fields, got: %+v", table.Rows[0])
		}
	})

	t.Run("预览行数限制", func(t *testing.T) {
		src := "h\n" + strings.Repeat("1\n", 30)
		table, err := imp.Parse(strings.NewReader(src), &Options{HasHeader: true, PreviewLimit: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(table.Rows) != 5 {
			t.Fatalf("preview limit not respected: got %d rows", len(table.Rows))
		}
	})
}

// TestForFilename 校验按扩展名选择解析器与不支持类型的显式报错（不支持的后缀绝不能静默返回 nil）。
func TestForFilename(t *testing.T) {
	if _, err := ForFilename("data.CSV"); err != nil {
		t.Fatalf("csv (case-insensitive) must be supported: %v", err)
	}
	if _, err := ForFilename("book.xlsx"); err != nil {
		t.Fatalf("xlsx must be supported: %v", err)
	}
	if _, err := ForFilename("archive.parquet"); err == nil {
		t.Fatal("unsupported extension must return error")
	}
}
