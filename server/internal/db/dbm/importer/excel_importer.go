package importer

import (
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"mayfly-go/pkg/errorx"

	"github.com/xuri/excelize/v2"
)

// ExcelImporter 解析 .xlsx 工作簿，默认读取第一个工作表（opts.Sheet 指定时读取对应表）。
//
// 读取单元格原始值（RawCellValue）以避免显示格式污染（千分位/百分比），但原始值会把日期单元格
// 还原成 Excel 序列数（如 45293.4375），因此对「数字且套了日期格式」的单元格做一次序列数→ISO 文本
// 的还原，保证日期/时间列拿到的是可读时间而非序列数。全空行不作为数据行。
type ExcelImporter struct{}

func (e *ExcelImporter) Extensions() []string { return []string{"xlsx", "xlsm"} }

func (e *ExcelImporter) Parse(r io.Reader, opts *Options) (*Table, error) {
	if opts == nil {
		opts = &Options{}
	}
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, errorx.NewBizf("read excel file failed: %s", err.Error())
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errorx.NewBiz("excel file has no worksheet")
	}
	sheet := opts.Sheet
	if sheet == "" {
		sheet = sheets[0]
	}

	rawRows, err := readExcelRows(f, sheet)
	if err != nil {
		return nil, errorx.NewBizf("read excel sheet [%s] failed: %s", sheet, err.Error())
	}

	table := &Table{Rows: make([][]string, 0, len(rawRows)), Sheets: sheets}
	if len(rawRows) == 0 {
		return table, nil
	}
	if opts.HasHeader {
		table.HasHeader = true
		table.Headers = rawRows[0]
		rawRows = rawRows[1:]
	}
	for _, row := range rawRows {
		if isBlankRow(row) {
			continue
		}
		if opts.PreviewLimit > 0 && len(table.Rows) >= opts.PreviewLimit {
			break
		}
		table.Rows = append(table.Rows, row)
	}
	return table, nil
}

// readExcelRows 以原始值模式逐行读取，并把日期/时间格式的数字单元格还原为 ISO 文本。
func readExcelRows(f *excelize.File, sheet string) ([][]string, error) {
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	rawOpts := excelize.Options{RawCellValue: true}
	var result [][]string
	rowNum := 0
	for rows.Next() {
		rowNum++
		cols, colErr := rows.Columns(rawOpts)
		if colErr != nil {
			return nil, colErr
		}
		normalizeDateCells(f, sheet, rowNum, cols)
		result = append(result, cols)
	}
	if err := rows.Error(); err != nil {
		return nil, err
	}
	return result, nil
}

// normalizeDateCells 对「值是数字」且「单元格套了日期/时间格式」的格子，把 Excel 序列数转成 ISO 文本。
// 非日期格式的纯数字（金额、数量等）保持原样，不受影响。
func normalizeDateCells(f *excelize.File, sheet string, rowNum int, cols []string) {
	for i, v := range cols {
		serial, ok := parseExcelSerial(v)
		if !ok {
			continue
		}
		ref, err := excelize.CoordinatesToCellName(i+1, rowNum)
		if err != nil {
			continue
		}
		if !cellHasDateTimeFormat(f, sheet, ref) {
			continue
		}
		cols[i] = excelSerialToTime(serial)
	}
}

// parseExcelSerial 解析落在合法日期区间的 Excel 序列数（1900 系统，1..≈9999-12-31）。
func parseExcelSerial(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || f < 1 || f > 2958466 { // 2958466 ≈ 9999-12-31
		return 0, false
	}
	return f, true
}

// excelSerialToTime 把 Excel 序列数按 1900 日期系统转为 ISO 文本：
// 整天无数值小数部分→"2006-01-02"，含时间→"2006-01-02 15:04:05"。
// 说明：Excel 1900 系统把 1900 误当闰年，序列 ≤60 会有一天偏差（1900-03-01 之前的日期，实际导入极罕见）；
// 1904 日期系统的工作簿不在此处理（绝大多数 .xlsx 为 1900 系统）。
func excelSerialToTime(serial float64) string {
	const excel1900Epoch = "1899-12-30"
	base, _ := time.Parse("2006-01-02", excel1900Epoch)
	days := int64(math.Floor(serial))
	frac := serial - float64(days)
	t := base.AddDate(0, 0, int(days))
	if frac <= 0 {
		return t.Format("2006-01-02")
	}
	t = t.Add(time.Duration(math.Round(frac * float64(24*time.Hour))))
	return t.Format("2006-01-02 15:04:05")
}

// builtinDateTimeNumFmts Excel 内置的日期/时间数字格式 ID。
var builtinDateTimeNumFmts = map[int]bool{
	14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true, 21: true, 22: true,
	27: true, 28: true, 29: true, 30: true, 31: true, 32: true, 33: true, 34: true, 35: true, 36: true,
	45: true, 46: true, 47: true, 50: true, 57: true, 58: true,
}

// cellHasDateTimeFormat 判定单元格的数字格式是否为日期/时间。
func cellHasDateTimeFormat(f *excelize.File, sheet, ref string) bool {
	styleID, err := f.GetCellStyle(sheet, ref)
	if err != nil || styleID == 0 {
		return false
	}
	st, err := f.GetStyle(styleID)
	if err != nil {
		return false
	}
	if builtinDateTimeNumFmts[st.NumFmt] {
		return true
	}
	if st.CustomNumFmt != nil {
		return looksLikeDateTimeFormat(*st.CustomNumFmt)
	}
	return false
}

// looksLikeDateTimeFormat 自定义格式启发式判定：去掉引号字面量后，含 y/m/d/h/s 或时间冒号即视为日期/时间格式。
// 数字格式（0 # ? % . , ; e E 及引号文本）不含这些字母，故不会误判纯数字单元格。
func looksLikeDateTimeFormat(format string) bool {
	var b strings.Builder
	inQuote := false
	for _, r := range format {
		if r == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote {
			b.WriteRune(r)
		}
	}
	s := strings.ToLower(b.String())
	return strings.ContainsAny(s, "ymdhs") || strings.Contains(s, ":")
}

func init() {
	Register(&ExcelImporter{})
}
