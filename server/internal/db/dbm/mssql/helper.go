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

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

// mssql 在insert语句前后不能识别begin和commit语句
func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) {
}

// mssql 在insert语句前后不能识别begin和commit语句
func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) {
}

func (dh *DumpHelper) BeforeInsertSql(quoteSchema string, tableName string) string {
	return fmt.Sprintf("set identity_insert %s.%s on ", quoteSchema, tableName)
}
