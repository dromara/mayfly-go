package oracle

import (
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
	// 如果默认值包含.nextval，说明是序列，默认值为null
	if strings.Contains(column.ColumnDefault, ".nextval") {
		column.ColumnDefault = ""
	}

	// 统一处理一下数据类型的长度
	if collx.ArrayAnyMatches([]string{"date", "time", "lob", "int"}, strings.ToLower(string(column.DataType))) {
		// 如果是不需要设置长度的类型
		column.CharMaxLength = 0
		column.NumPrecision = 0
	} else if strings.Contains(strings.ToLower(string(column.DataType)), "char") {
		// 如果是字符串类型，长度最大4000，否则修改字段类型为clob
		if column.CharMaxLength > 4000 {
			column.DataType = "NCLOB"
			column.CharMaxLength = 0
			column.NumPrecision = 0
		}
	}
}
