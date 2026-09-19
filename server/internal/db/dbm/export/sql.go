package export

import (
	"io"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
)

// ========== SQLConsumer：SQL 格式导出消费者 ==========
//
// 生成方言感知的 INSERT 语句，支持事务控制（通过 DumpHelper）。
// 方言差异完全委托给 DumpHelper + SQLGenerator，本消费者不含任何方言硬编码。

// SQLConsumer SQL 格式导出消费者。
type SQLConsumer struct{}

// 编译期接口断言
var _ Consumer = (*SQLConsumer)(nil)

func (c *SQLConsumer) Format() string { return "sql" }

func (c *SQLConsumer) SupportsScript() bool { return true }

// Name 格式展示名（前端按 Format 键做 i18n，此处为规范英文名）
func (c *SQLConsumer) Name() string          { return "SQL Script" }
func (c *SQLConsumer) ContentType() string   { return "text/sql" }
func (c *SQLConsumer) FileExtension() string { return ".sql" }

func (c *SQLConsumer) Begin(w io.Writer, tableName string, columns []dbi.Column, settings *Settings) error {
	return nil
}

func (c *SQLConsumer) ConsumeBatch(w io.Writer, tableName string, columns []dbi.Column, rows [][]any,
	helper dbi.DumpHelper, sqlGen dbi.SQLGenerator, settings *Settings) error {
	if len(rows) == 0 {
		return nil
	}

	// 方言前置钩子（如 MSSQL/DM 的 SET IDENTITY_INSERT ON）
	if helper != nil {
		beforeInsert := helper.BeforeInsertSql(tableName, columns)
		if beforeInsert != "" {
			if _, err := io.WriteString(w, beforeInsert); err != nil {
				return err
			}
		}
	}

	// 方言感知的 INSERT 生成
	insertSql := sqlGen.GenInsert(tableName, columns, rows, dbi.DuplicateStrategyNone, nil)
	if _, err := io.WriteString(w, strings.Join(insertSql, ";\n")+";\n"); err != nil {
		return err
	}
	return nil
}

func (c *SQLConsumer) End(w io.Writer, tableName string, settings *Settings) error {
	return nil
}

// Finish SQL 无全局收尾（事务闭合由 DumpHelper.AfterInsert 按表处理）
func (c *SQLConsumer) Finish(w io.Writer, settings *Settings) error { return nil }

func init() {
	Register(&SQLConsumer{})
}
