package mssql

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

var (
	columnHelper = &ColumnHelper{}
)

type ColumnHelper struct {
}

func (ch *ColumnHelper) FixColumn(column *dbi.Column) {
	dataType := strings.ToLower(string(column.DataType))

	if collx.ArrayAnyMatches([]string{"date", "time"}, dataType) {
		// 如果是datetime，精度取NumScale字段
		column.CharMaxLength = column.NumScale
	} else if collx.ArrayAnyMatches([]string{"int", "bit", "real", "text", "xml"}, dataType) {
		// 不显示长度的类型
		column.NumPrecision = 0
		column.CharMaxLength = 0
	} else if collx.ArrayAnyMatches([]string{"numeric", "decimal", "float"}, dataType) {
		// 如果是num，长度取精度和小数位数
		column.CharMaxLength = 0
	} else if collx.ArrayAnyMatches([]string{"nvarchar", "nchar"}, dataType) {
		// 如果是nvarchar，可视长度减半
		column.CharMaxLength = column.CharMaxLength / 2
	}

	if collx.ArrayAnyMatches([]string{"char"}, dataType) {
		// char最大长度4000
		if column.CharMaxLength >= 4000 {
			column.DataType = "ntext"
			column.CharMaxLength = 0
		}
	}

}

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

// mssql 在insert语句前后不能识别begin和commit语句
func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

// mssql 在insert语句前后不能识别begin和commit语句；无自增列时也不输出COMMIT
func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) error {
	// 对应BeforeInsertSql输出的identity_insert on，需输出off，否则会话保持on状态，
	// 后续其他含自增表的on语句会报"already ON for table"错误
	if hasIdentityColumn(columns) {
		_, err := writer.Write([]byte(fmt.Sprintf("set identity_insert %s off;\n", mssqlQuoter.QuoteIdent(tableName))))
		return err
	}
	return nil
}

// 仅含自增列的表才需要set identity_insert，否则mssql报"does not have the identity property"错误
func (dh *DumpHelper) BeforeInsertSql(tableName string, columns []dbi.Column) string {
	if !hasIdentityColumn(columns) {
		return ""
	}
	return fmt.Sprintf("set identity_insert %s on;\n", mssqlQuoter.QuoteIdent(tableName))
}
