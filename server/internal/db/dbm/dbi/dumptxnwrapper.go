package dbi

import "io"

// DumpTxnWrapper 导出（dump）辅助方法：在数据行插入前后写方言特定的包装语句。
// 各库差异大（如 pg 不输出 BEGIN/COMMIT、mssql/dm 需 set identity_insert），故抽象为可选覆写点。
type DumpTxnWrapper interface {
	// BeforeInsert 每表数据插入开始前的前置输出
	BeforeInsert(writer io.Writer, tableName string) error

	// BeforeInsertSQL 生成每批insert语句前的前置语句（如mssql/dm的set identity_insert on）
	// - tableName为裸表名，由各方言helper自行quote（避免调用方使用源方言引用符）
	// - columns用于判断表是否含自增列（对无自增列的表set identity_insert会报错）
	BeforeInsertSQL(tableName string, columns []Column) string

	// AfterInsert 每表数据插入完成后的收尾输出
	AfterInsert(writer io.Writer, tableName string, columns []Column) error
}

// DefaultDumpTxnWrapper DumpTxnWrapper 默认实现：标准 BEGIN/COMMIT 包装，无前置语句。
type DefaultDumpTxnWrapper struct {
}

func (dd *DefaultDumpTxnWrapper) BeforeInsert(writer io.Writer, tableName string) error {
	_, err := writer.Write([]byte("BEGIN;\n"))
	return err
}

func (dd *DefaultDumpTxnWrapper) BeforeInsertSQL(tableName string, columns []Column) string {
	return ""
}

func (dd *DefaultDumpTxnWrapper) AfterInsert(writer io.Writer, tableName string, columns []Column) error {
	_, err := writer.Write([]byte("COMMIT;\n"))
	return err
}
