package importer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestCSVStreamYieldsHeaderAndRows 流式接口按「表头一次 + 每数据行一次」回调，
// BOM 剥离、全空行跳过与 Parse 同口径；且不聚合整表（由调用方决定是否累积）。
func TestCSVStreamYieldsHeaderAndRows(t *testing.T) {
	var header, sheets []string
	var rows [][]string
	err := (&CSVImporter{}).Stream(strings.NewReader("\ufeffid,name\n1,a\n,,\n2,b\n"), &Options{HasHeader: true}, &StreamHandler{
		OnHeader: func(h []string, s []string) error { header, sheets = h, s; return nil },
		OnRow:    func(r []string) error { rows = append(rows, append([]string(nil), r...)); return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(header) == 0 || header[0] != "id" {
		t.Fatalf("BOM must be stripped from header, got %q", header)
	}
	if sheets != nil {
		t.Fatalf("csv sheets must be nil, got %v", sheets)
	}
	if len(rows) != 2 {
		t.Fatalf("blank row must be skipped: want 2 rows, got %d (%q)", len(rows), rows)
	}
}

// TestCSVStreamNoHeader 无表头模式：OnHeader 仍回调一次（headers 为 nil），数据行全量回调。
func TestCSVStreamNoHeader(t *testing.T) {
	headerCalled := false
	var header []string
	rows := 0
	err := (&CSVImporter{}).Stream(strings.NewReader("1,a\n2,b\n"), &Options{}, &StreamHandler{
		OnHeader: func(h []string, _ []string) error { headerCalled = true; header = h; return nil },
		OnRow:    func(r []string) error { rows++; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !headerCalled || header != nil {
		t.Fatalf("OnHeader must be called once with nil headers, got called=%v header=%v", headerCalled, header)
	}
	if rows != 2 {
		t.Fatalf("want 2 rows, got %d", rows)
	}
}

// TestExcelStreamYieldsRows Excel 流式接口按行回调，表头为 sheet 首行。
func TestExcelStreamYieldsRows(t *testing.T) {
	f := excelize.NewFile()
	const sheet = "Sheet1"
	_ = f.SetCellValue(sheet, "A1", "id")
	_ = f.SetCellValue(sheet, "A2", 1)
	_ = f.SetCellValue(sheet, "A3", 2)
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}

	var header []string
	rows := 0
	err = (&ExcelImporter{}).Stream(bytes.NewReader(buf.Bytes()), &Options{HasHeader: true}, &StreamHandler{
		OnHeader: func(h []string, _ []string) error { header = h; return nil },
		OnRow:    func(r []string) error { rows++; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(header) == 0 || header[0] != "id" {
		t.Fatalf("bad header: %q", header)
	}
	if rows != 2 {
		t.Fatalf("want 2 rows, got %d", rows)
	}
}

// TestParsePreviewStopsEarly Parse 仍是预览入口：PreviewLimit 在流式迭代中即时生效。
func TestParsePreviewStopsEarly(t *testing.T) {
	table, err := (&CSVImporter{}).Parse(strings.NewReader("id\n1\n2\n3\n4\n5\n"), &Options{HasHeader: true, PreviewLimit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("preview must stop at 2, got %d", len(table.Rows))
	}
}
