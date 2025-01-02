package postgres

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/dbm/dbi"
	"regexp"
	"strings"
)

var (
	// 提取pg默认值， 如：'id'::varchar  提取id  ；  '-1'::integer  提取-1
	defaultValueRegexp = regexp.MustCompile(`'([^']*)'`)
)

func FixColumnDefault(column *dbi.Column) {
	// 如果默认值带冒号，如：'id'::varchar
	if column.ColumnDefault != "" && strings.Contains(column.ColumnDefault, "::") && !strings.HasPrefix(column.ColumnDefault, "nextval") {
		match := defaultValueRegexp.FindStringSubmatch(column.ColumnDefault)
		if len(match) > 1 {
			column.ColumnDefault = match[1]
		}
	}
}

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) {
	// 设置自增序列当前值
	for _, column := range columns {
		if column.IsIdentity {
			seq := fmt.Sprintf("SELECT setval('%s_%s_seq', (SELECT max(%s) FROM %s));\n", tableName, column.ColumnName, column.ColumnName, tableName)
			writer.Write([]byte(seq))
		}
	}

	writer.Write([]byte("COMMIT;\n"))
}
