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

// Parse 预览/小样本解析：委托 Stream 累积前 PreviewLimit 行（<=0 不限制）
func (c *CSVImporter) Parse(r io.Reader, opts *Options) (*Table, error) {
	return parseTable(r, c, opts)
}

// Stream 逐行读取并回调，不在解析层聚合整表（供全量导入端到端流式消费）。
func (c *CSVImporter) Stream(r io.Reader, opts *Options, h *StreamHandler) error {
	if opts == nil {
		opts = &Options{}
	}
	cr := csv.NewReader(r)
	if sep := csvSeparator(opts.FieldSeparator); sep != 0 {
		cr.Comma = sep
	}
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true

	headerSent := false
	emitHeader := func(headers []string) error {
		if headerSent {
			return nil
		}
		headerSent = true
		if h != nil && h.OnHeader != nil {
			return h.OnHeader(headers, nil)
		}
		return nil
	}

	rowNum := 0
	for {
		rec, readErr := cr.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return errorx.NewBizf("parse csv file failed: %s", readErr.Error())
		}
		rowNum++
		// 剥离首行首格的 UTF-8 BOM：它会粘在首个单元格上（表头名或首行值），
		// 使表头匹配失败、或让数值列首格变成带隐形字符的非法值
		if rowNum == 1 && len(rec) > 0 {
			rec[0] = strings.TrimPrefix(rec[0], utf8BOM)
		}
		if rowNum == 1 {
			var headers []string
			if opts.HasHeader {
				headers = rec
			}
			if err := emitHeader(headers); err != nil {
				return err
			}
			if opts.HasHeader {
				continue
			}
		}
		if isBlankRow(rec) {
			continue
		}
		if h != nil && h.OnRow != nil {
			if err := h.OnRow(rec); err != nil {
				return err
			}
		}
	}
	// 空文件也要回调一次表头，让调用方拿到「无表头」的确定结果（而不是永远等不到回调）
	return emitHeader(nil)
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
