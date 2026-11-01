package importer

import (
	"fmt"
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

// Parse 预览/小样本解析：委托 Stream 累积前 PreviewLimit 行（<=0 不限制）
func (e *ExcelImporter) Parse(r io.Reader, opts *Options) (*Table, error) {
	return parseTable(r, e, opts)
}

// Stream 逐行解析并回调，不在解析层聚合整张 sheet（供全量导入端到端流式消费）。
func (e *ExcelImporter) Stream(r io.Reader, opts *Options, h *StreamHandler) error {
	if opts == nil {
		opts = &Options{}
	}
	f, err := excelize.OpenReader(r)
	if err != nil {
		return errorx.NewBizf("read excel file failed: %s", err.Error())
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return errorx.NewBiz("excel file has no worksheet")
	}
	sheet := opts.Sheet
	if sheet == "" {
		sheet = sheets[0]
	}

	headerSent := false
	emitHeader := func(headers []string) error {
		if headerSent {
			return nil
		}
		headerSent = true
		if h != nil && h.OnHeader != nil {
			return h.OnHeader(headers, sheets)
		}
		return nil
	}

	first := true
	err = scanExcelRows(f, sheet, dateSerialColumnSet(opts.DateSerialColumns), func(cols []string) error {
		if first {
			first = false
			var headers []string
			if opts.HasHeader {
				headers = cols
			}
			if err := emitHeader(headers); err != nil {
				return err
			}
			if opts.HasHeader {
				return nil
			}
		}
		if isBlankRow(cols) {
			return nil
		}
		if h != nil && h.OnRow != nil {
			return h.OnRow(cols)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 空表也要回调一次表头，让调用方拿到「无表头」的确定结果
	return emitHeader(nil)
}

// scanExcelRows 以原始值模式逐行读取并回调，日期/时间格式的数字单元格先还原为 ISO 文本。
// 回调返回错误即中止扫描（调用方用 sentinel 区分「提前结束」与真实错误）。
func scanExcelRows(f *excelize.File, sheet string, dateCols map[int]bool, fn func(cols []string) error) error {
	rows, err := f.Rows(sheet)
	if err != nil {
		return errorx.NewBizf("read excel sheet [%s] failed: %s", sheet, err.Error())
	}
	defer func() { _ = rows.Close() }()

	rawOpts := excelize.Options{RawCellValue: true}
	for rows.Next() {
		cols, colErr := rows.Columns(rawOpts)
		if colErr != nil {
			return errorx.NewBizf("read excel sheet [%s] failed: %s", sheet, colErr.Error())
		}
		normalizeDateCols(cols, dateCols)
		if err := fn(cols); err != nil {
			return err
		}
	}
	if err := rows.Error(); err != nil {
		return errorx.NewBizf("read excel sheet [%s] failed: %s", sheet, err.Error())
	}
	return nil
}

// dateSerialColumnSet 把列下标切片转成集合（空切片返回 nil，热路径零分配）
func dateSerialColumnSet(indices []int) map[int]bool {
	if len(indices) == 0 {
		return nil
	}
	set := make(map[int]bool, len(indices))
	for _, i := range indices {
		if i >= 0 {
			set[i] = true
		}
	}
	return set
}

// normalizeDateCols 把指定列（目标列为日期/时间）的 Excel 序列数还原为时间文本。
//
// 判据是"这一列映射到日期/时间目标列"，而不是"该单元格套了日期格式"：后者必须逐格
// GetCellStyle，会让 excelize 全量解析整张 sheet。代价是：目标列是字符串时，日期单元格
// 不再转 ISO（保留序列数），这是为内存安全做的显式取舍。
func normalizeDateCols(cols []string, dateCols map[int]bool) {
	if len(dateCols) == 0 {
		return
	}
	for i := range cols {
		if !dateCols[i] {
			continue
		}
		serial, ok := parseExcelSerial(cols[i])
		if !ok {
			continue
		}
		cols[i] = excelSerialToTime(serial)
	}
}

// parseExcelSerial 解析落在合法日期区间的 Excel 序列数（1900 系统，1..≈9999-12-31）。
func parseExcelSerial(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	// 允许 [0,1)：纯时间单元格（12:00 = 0.5）没有整数部分，旧实现整体丢弃，
	// 写 time 列会变成 "0.5" 这类非法文本。
	if err != nil || f < 0 || f > 2958466 { // 2958466 ≈ 9999-12-31
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
	// 无整数部分 = 纯时间（12:00 存成 0.5），输出 HH:MM:SS
	if days == 0 {
		// 输出 SQL time 字面量形态（HH:MM:SS），不是 Duration.String()
		d := time.Duration(math.Round(frac * float64(24*time.Hour)))
		return fmt.Sprintf("%02d:%02d:%02d", int(d/time.Hour), int(d%time.Hour/time.Minute), int(d%time.Minute/time.Second))
	}
	t := base.AddDate(0, 0, int(days))
	if frac <= 0 {
		return t.Format("2006-01-02")
	}
	t = t.Add(time.Duration(math.Round(frac * float64(24*time.Hour))))
	return t.Format("2006-01-02 15:04:05")
}

func init() {
	Register(&ExcelImporter{})
}
