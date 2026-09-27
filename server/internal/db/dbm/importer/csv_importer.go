package importer

import (
	"encoding/csv"
	"io"
	"strings"

	"mayfly-go/pkg/errorx"
)

// utf8BOM UTF-8 字节序标记，前端/Excel 导出的 CSV 常在文件首字节带上它。
const utf8BOM = "\ufeff"

// CSVImporter 解析 RFC 4180 形态的 CSV/TSV 文本文件。
//
// 与 export 包的 CSVConsumer 对称：导出用 csv.Writer，导入用 csv.Reader。
// 开启 LazyQuotes 以容忍导出/手工编辑产物中裸引号等非严格写法；FieldsPerRecord=-1 允许不等宽行，
// 交由上层按列位置映射对齐。全空行（连续/尾随空行）不作为数据行。
type CSVImporter struct{}

func (c *CSVImporter) Extensions() []string { return []string{"csv", "tsv", "txt"} }

func (c *CSVImporter) Parse(r io.Reader, opts *Options) (*Table, error) {
	if opts == nil {
		opts = &Options{}
	}
	cr := csv.NewReader(r)
	if sep := csvSeparator(opts.FieldSeparator); sep != 0 {
		cr.Comma = sep
	}
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true

	records, err := cr.ReadAll()
	if err != nil {
		return nil, errorx.NewBizf("parse csv file failed: %s", err.Error())
	}

	table := &Table{Rows: make([][]string, 0, len(records))}
	if len(records) == 0 {
		return table, nil
	}
	// 剥离文件首字节的 UTF-8 BOM：它会粘在首个单元格上（表头名或首行值），
	// 使表头匹配失败、或让数值列首格变成带隐形字符的非法值
	if len(records[0]) > 0 {
		records[0][0] = strings.TrimPrefix(records[0][0], utf8BOM)
	}
	if opts.HasHeader {
		table.HasHeader = true
		table.Headers = records[0]
		records = records[1:]
	}
	for _, rec := range records {
		if isBlankRow(rec) {
			continue
		}
		if opts.PreviewLimit > 0 && len(table.Rows) >= opts.PreviewLimit {
			break
		}
		table.Rows = append(table.Rows, rec)
	}
	return table, nil
}

// csvSeparator 解析分隔符配置：显式配置取其首字符；\t/tab 归一为制表符；未配置返回 0（沿用 csv 默认逗号）。
func csvSeparator(s string) rune {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return 0
	case `\t`, "tab", "\t":
		return '\t'
	default:
		for _, rn := range s {
			return rn
		}
		return 0
	}
}

func init() {
	Register(&CSVImporter{})
}
