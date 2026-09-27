package importer

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// TestExcelDateRoundtrip 验证日期单元格不再落成 Excel 序列数：写一个带日期格式的 time 单元格，
// 解析回来应是可读 ISO 文本；同表的纯数字单元格不受影响。
func TestExcelDateRoundtrip(t *testing.T) {
	f := excelize.NewFile()
	const sheet = "Sheet1"
	for i, h := range []string{"name", "amount", "dt"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	_ = f.SetCellValue(sheet, "A2", "alice")
	_ = f.SetCellValue(sheet, "B2", 1234.5)
	_ = f.SetCellValue(sheet, "C2", time.Date(2024, 1, 2, 10, 30, 0, 0, time.UTC))
	style, err := f.NewStyle(&excelize.Style{NumFmt: 14}) // 内置日期格式
	if err != nil {
		t.Fatal(err)
	}
	_ = f.SetCellStyle(sheet, "C2", "C2", style)

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}

	table, err := (&ExcelImporter{}).Parse(bytes.NewReader(buf.Bytes()), &Options{HasHeader: true})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(table.Rows) != 1 {
		t.Fatalf("want 1 data row, got %d", len(table.Rows))
	}
	row := table.Rows[0]
	if len(row) < 3 {
		t.Fatalf("row too short: %q", row)
	}
	if row[1] != "1234.5" {
		t.Errorf("纯数字单元格应保持原值，got %q", row[1])
	}
	if row[2] != "2024-01-02 10:30:00" {
		t.Errorf("日期单元格应还原为 ISO 文本，got %q", row[2])
	}
}

// TestExcelSkipBlankRows 验证全空行（Excel 尾随/中间空白行）不作为数据行。
func TestExcelSkipBlankRows(t *testing.T) {
	f := excelize.NewFile()
	const sheet = "Sheet1"
	_ = f.SetCellValue(sheet, "A1", "id")
	_ = f.SetCellValue(sheet, "A2", "1")
	_ = f.SetCellValue(sheet, "A4", "2") // A3 留空行
	buf, _ := f.WriteToBuffer()

	table, err := (&ExcelImporter{}).Parse(bytes.NewReader(buf.Bytes()), &Options{HasHeader: true})
	if err != nil {
		t.Fatal(err)
	}
	// 空行可能被 excelize 返回为空切片，应被跳过；有效行为 1、2
	for _, r := range table.Rows {
		if isBlankRow(r) {
			t.Errorf("空行未被过滤: %q", r)
		}
	}
}

// TestCSVBomAndBlankRows 验证 CSV 首字节 BOM 被剥离、全空行被跳过。
func TestCSVBomAndBlankRows(t *testing.T) {
	// 有表头：BOM 粘在首个表头名上
	table, err := (&CSVImporter{}).Parse(strings.NewReader("\ufeffid,name\n1,alice\n,,\n2,bob\n"), &Options{HasHeader: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Headers) == 0 || table.Headers[0] != "id" {
		t.Fatalf("BOM 未剥离，headers=%q", table.Headers)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("全空行应被跳过，got %d rows: %q", len(table.Rows), table.Rows)
	}

	// 无表头：BOM 粘在首个数据值上，数值列首格不得带隐形字符
	table2, err := (&CSVImporter{}).Parse(strings.NewReader("\ufeff123,abc\n"), &Options{})
	if err != nil {
		t.Fatal(err)
	}
	if table2.Rows[0][0] != "123" {
		t.Fatalf("无表头 BOM 未剥离，first cell=%q", table2.Rows[0][0])
	}
}
