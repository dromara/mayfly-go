package mssql

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

var _ dbi.DumpTxnWrapper = (*DumpTxnWrapper)(nil)

type DumpTxnWrapper struct {
	dbi.DefaultDumpTxnWrapper
}

// hasIdentityColumn 判断表是否包含自增列
func hasIdentityColumn(columns []dbi.Column) bool {
	for _, col := range columns {
		if col.AutoIncrement {
			return true
		}
	}
	return false
}

// mssql 在insert语句前后不能识别begin和commit语句
func (dh *DumpTxnWrapper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

// mssql 在insert语句前后不能识别begin和commit语句；无自增列时也不输出COMMIT
func (dh *DumpTxnWrapper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) error {
	// 对应BeforeInsertSQL输出的identity_insert on，需输出off，否则会话保持on状态，
	// 后续其他含自增表的on语句会报"already ON for table"错误
	if hasIdentityColumn(columns) {
		_, err := writer.Write([]byte(fmt.Sprintf("set identity_insert %s off;\n", mssqlQuoter.QuoteIdent(tableName))))
		return err
	}
	return nil
}

// 仅含自增列的表才需要set identity_insert，否则mssql报"does not have the identity property"错误
func (dh *DumpTxnWrapper) BeforeInsertSQL(tableName string, columns []dbi.Column) string {
	if !hasIdentityColumn(columns) {
		return ""
	}
	return fmt.Sprintf("set identity_insert %s on;\n", mssqlQuoter.QuoteIdent(tableName))
}
