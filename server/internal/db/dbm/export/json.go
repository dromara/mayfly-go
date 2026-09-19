package export

import (
	"encoding/json"
	"io"

	"mayfly-go/internal/db/dbm/dbi"
)

// ========== JSONConsumer：JSON 格式导出消费者 ==========
//
// 方言无关的通用 JSON 序列化。产物结构随导出表数自适应（由 Settings.TableCount 决定）：
//   - 单表：纯记录数组   [{...}, {...}]
//   - 多表：以表名为键的对象 {"t1": [{...}], "t2": [{...}]}
//     ——若多表仍逐表输出顶层数组，拼接产物是非法 JSON（两个相邻数组值），消费方无法解析
//
// 表数在 Begin 前由 Exporter 注入，结构模式在首个 Begin 时锁定；Begin 早于任何
// ConsumeBatch/End，锁定时机必然先于内容写入，无状态错位。
// 注意：Exporter 按表串行调用 Begin/ConsumeBatch/End，结束后调 Finish，无并发问题。

// JSONConsumer JSON 格式导出消费者。
type JSONConsumer struct {
	firstRow  bool // 当前表是否尚未写入任何行
	anyTable  bool // 是否已有表开始输出（多表对象模式下决定键前是否加逗号）
	multiMode bool // 产物结构模式：true=多表对象，在首个 Begin 按 TableCount 锁定
}

// 编译期接口断言
var _ Consumer = (*JSONConsumer)(nil)

func (c *JSONConsumer) Format() string { return "json" }

func (c *JSONConsumer) SupportsScript() bool { return false }

func (c *JSONConsumer) Name() string          { return "JSON" }
func (c *JSONConsumer) ContentType() string   { return "application/json" }
func (c *JSONConsumer) FileExtension() string { return ".json" }

func (c *JSONConsumer) Begin(w io.Writer, tableName string, columns []dbi.Column, settings *Settings) error {
	if !c.anyTable {
		// 首个表：按表总数锁定产物结构
		c.multiMode = settings != nil && settings.TableCount > 1
		if c.multiMode {
			if _, err := io.WriteString(w, "{\n"); err != nil {
				return err
			}
		} else {
			if _, err := io.WriteString(w, "[\n"); err != nil {
				return err
			}
		}
	}
	if c.multiMode {
		// 表间以逗号分隔；键必须经 JSON 转义（表名可含引号/换行等特殊字符）
		sep := "\n"
		if c.anyTable {
			sep = ",\n"
		}
		key, err := json.Marshal(tableName)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(w, sep+string(key)+": [\n"); err != nil {
			return err
		}
	}
	c.anyTable = true
	c.firstRow = true
	return nil
}

func (c *JSONConsumer) ConsumeBatch(w io.Writer, tableName string, columns []dbi.Column, rows [][]any,
	helper dbi.DumpHelper, sqlGen dbi.SQLGenerator, settings *Settings) error {

	pretty := settings != nil && settings.PrettyPrint
	for _, row := range rows {
		obj := make(map[string]any, len(columns))
		for j, col := range columns {
			if j < len(row) {
				obj[col.ColumnName] = row[j]
			}
		}

		var data []byte
		var err error
		if pretty {
			data, err = json.MarshalIndent(obj, "", "  ")
		} else {
			data, err = json.Marshal(obj)
		}
		if err != nil {
			return err
		}

		// 非首行时前置逗号（正确处理跨批次逗号）
		if !c.firstRow {
			if _, err := io.WriteString(w, ",\n"); err != nil {
				return err
			}
		}
		c.firstRow = false

		if _, err := w.Write(data); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}

// End 闭合当前表的记录数组（多表对象模式下外层花括号由 Finish 闭合）
func (c *JSONConsumer) End(w io.Writer, tableName string, settings *Settings) error {
	_, err := io.WriteString(w, "]\n")
	return err
}

// Finish 多表对象模式闭合最外层花括号；单表数组模式无额外输出
func (c *JSONConsumer) Finish(w io.Writer, settings *Settings) error {
	if c.multiMode {
		_, err := io.WriteString(w, "}\n")
		return err
	}
	return nil
}

func init() {
	RegisterFactory("json", func() Consumer {
		return &JSONConsumer{}
	})
}
