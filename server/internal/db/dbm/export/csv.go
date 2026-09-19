package export

import (
	"encoding/csv"
	"fmt"
	"io"

	"mayfly-go/internal/db/dbm/dbi"
)

// ========== CSVConsumer：CSV 格式导出消费者 ==========
//
// 方言无关的通用 CSV 序列化，支持可配置分隔符、引号策略、编码等。
// 所有方言共享同一实现，无需任何方言适配代码。
//
// 注意：csv.Writer 使用内部缓冲区，必须在 End 时调用 Flush 确保数据写入底层 io.Writer。
// 因此 CSVConsumer 持有 cw 字段，在 Begin 时创建，ConsumeBatch 复用，End 时 flush。
// Exporter 按表串行调用 Begin/ConsumeBatch/End，无并发问题。

// CSVConsumer CSV 格式导出消费者。
type CSVConsumer struct {
	cw *csv.Writer // 在 Begin 时创建，ConsumeBatch 复用，End 时 flush
}

// 编译期接口断言
var _ Consumer = (*CSVConsumer)(nil)

func (c *CSVConsumer) Format() string { return "csv" }

func (c *CSVConsumer) SupportsScript() bool { return false }

func (c *CSVConsumer) Name() string          { return "CSV" }
func (c *CSVConsumer) ContentType() string   { return "text/csv" }
func (c *CSVConsumer) FileExtension() string { return ".csv" }

func (c *CSVConsumer) Begin(w io.Writer, tableName string, columns []dbi.Column, settings *Settings) error {
	c.cw = newCSVWriter(w, settings)
	if settings != nil && !settings.IncludeHeader {
		return nil
	}
	names := make([]string, len(columns))
	for i, col := range columns {
		names[i] = col.ColumnName
	}
	return c.cw.Write(names)
}

func (c *CSVConsumer) ConsumeBatch(w io.Writer, tableName string, columns []dbi.Column, rows [][]any,
	helper dbi.DumpHelper, sqlGen dbi.SQLGenerator, settings *Settings) error {
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, val := range row {
			record[i] = formatValue(val)
		}
		if err := c.cw.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func (c *CSVConsumer) End(w io.Writer, tableName string, settings *Settings) error {
	// csv.Writer 使用内部缓冲，必须显式 Flush 确保数据写入底层 io.Writer
	c.cw.Flush()
	return c.cw.Error()
}

// Finish CSV 无全局收尾（每表 End 已 flush 落盘）
func (c *CSVConsumer) Finish(w io.Writer, settings *Settings) error { return nil }

// newCSVWriter 创建按 settings 配置的 csv.Writer
func newCSVWriter(w io.Writer, settings *Settings) *csv.Writer {
	cw := csv.NewWriter(w)
	if settings != nil {
		if settings.FieldSeparator != "" && len(settings.FieldSeparator) > 0 {
			cw.Comma = rune(settings.FieldSeparator[0])
		}
		if settings.LineTerminator != "" && settings.LineTerminator != "\n" {
			cw.UseCRLF = settings.LineTerminator == "\r\n"
		}
	}
	return cw
}

// formatValue 将任意值格式化为字符串（CSV/JSON 共用）
func formatValue(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func init() {
	// CSV 消费者持有 cw 状态，每次获取需创建新实例
	RegisterFactory("csv", func() Consumer {
		return &CSVConsumer{}
	})
}
