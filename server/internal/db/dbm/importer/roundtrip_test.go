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

	// 日期序列数还原由**目标列**驱动：不指定 DateSerialColumns 时保持原始序列数，
	// 指定后才转 ISO（解析层不再逐格探测单元格格式，避免 excelize 全量解析整张 sheet）
	table, err := (&ExcelImporter{}).Parse(bytes.NewReader(buf.Bytes()), &Options{HasHeader: true, DateSerialColumns: []int{2}})
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

// TestExcelDateSerialOnlyForMappedColumns 只有被指定为日期列的列才做序列数还原。
//
// 这是 P1-6 的核心不变量：解析层不调用 GetCellStyle（那会让 excelize 把整张 sheet
// 反序列化常驻内存），所以"单元格套了日期格式"不再自动转 ISO —— 由上层按目标列类型决定。
func TestExcelDateSerialOnlyForMappedColumns(t *testing.T) {
	f := excelize.NewFile()
	const sheet = "Sheet1"
	_ = f.SetCellValue(sheet, "A1", "id")
	_ = f.SetCellValue(sheet, "A2", "1")
	_ = f.SetCellValue(sheet, "B1", "amount")
	_ = f.SetCellValue(sheet, "B2", "1234.5")
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}

	// 对照组：未指定日期列 → 纯数字列保持原值（金额、数量等不会被误当日期）
	table, err := (&ExcelImporter{}).Parse(bytes.NewReader(buf.Bytes()), &Options{HasHeader: true})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got := table.Rows[0][1]; got != "1234.5" {
		t.Errorf("未指定日期列时数值应保持原样，got %q", got)
	}

	// 指定为日期列 → 按 Excel 1900 序列数还原
	table, err = (&ExcelImporter{}).Parse(bytes.NewReader(buf.Bytes()), &Options{HasHeader: true, DateSerialColumns: []int{1}})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got := table.Rows[0][1]; got != "1903-05-18 12:00:00" {
		t.Errorf("指定为日期列后应按序列数还原，got %q", got)
	}
}

// TestExcelTimeOnlySerial Excel 纯时间单元格（序列 <1）必须还原为 HH:MM:SS。
//
// 旧实现以 serial < 1 拒绝，导致 12:00（0.5）被原样写成 "0.5"：写 TIME 列会报 1292，
// 或按整数解释成 00:00:00。
func TestExcelTimeOnlySerial(t *testing.T) {
	if got := excelSerialToTime(0.5); got != "12:00:00" {
		t.Errorf("0.5 应为 12:00:00，got %q", got)
	}
	if got := excelSerialToTime(0.25); got != "06:00:00" {
		t.Errorf("0.25 应为 06:00:00，got %q", got)
	}
	if _, ok := parseExcelSerial("0.5"); !ok {
		t.Error("纯时间序列 0.5 应被接受")
	}
	if _, ok := parseExcelSerial("-1"); ok {
		t.Error("负数不应被当作序列数")
	}
}
