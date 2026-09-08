package dm

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

var _ dbi.DumpHelper = (*DumpHelper)(nil)

type DumpHelper struct {
	dbi.DefaultDumpHelper
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

func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

// dm不输出BEGIN/COMMIT包装（与BeforeInsert的空实现对称），有自增列时输出identity_insert off
func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) error {
	// 对应BeforeInsertSql输出的identity_insert on，需输出off，否则会话保持on状态，
	// 后续其他含自增表的on语句会报"already ON for table"错误
	if hasIdentityColumn(columns) {
		_, err := writer.Write([]byte(fmt.Sprintf("set identity_insert %s off;\n", dbi.DefaultQuoter.QuoteIdent(tableName))))
		return err
	}
	return nil
}

// 仅含自增列的表才需要set identity_insert，否则达梦报错；表名需引用避免保留字冲突
func (dh *DumpHelper) BeforeInsertSql(tableName string, columns []dbi.Column) string {
	if !hasIdentityColumn(columns) {
		return ""
	}
	return fmt.Sprintf("set identity_insert %s on;\n", dbi.DefaultQuoter.QuoteIdent(tableName))
}
