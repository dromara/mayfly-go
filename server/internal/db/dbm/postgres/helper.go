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
	// 默认值本身可能含单引号（如 'it''s'::varchar），需支持双写引号
	defaultValueRegexp = regexp.MustCompile(`'((?:[^']|'')*)'`)
)

func FixColumnDefault(column *dbi.Column) {
	// 如果默认值带冒号，如：'id'::varchar
	if column.ColumnDefault != "" && strings.Contains(column.ColumnDefault, "::") && !strings.HasPrefix(column.ColumnDefault, "nextval") {
		match := defaultValueRegexp.FindStringSubmatch(column.ColumnDefault)
		if len(match) > 1 {
			// 双写引号还原为单引号
			column.ColumnDefault = strings.ReplaceAll(match[1], "''", "'")
		}
	}
}

type DumpHelper struct {
	dbi.DefaultDumpHelper
}

// pg导入方（transfer2Db/ExecReader）已在自身事务内逐条执行，脚本内的BEGIN/COMMIT语句
// 会提交/破坏外层事务（报 unexpected transaction status idle），故不输出，与sqlite/mssql保持一致
func (dh *DumpHelper) BeforeInsert(writer io.Writer, tableName string) error {
	return nil
}

func (dh *DumpHelper) AfterInsert(writer io.Writer, tableName string, columns []dbi.Column) error {
	// 设置自增序列当前值
	for _, column := range columns {
		if column.AutoIncrement {
			seq := fmt.Sprintf("SELECT setval('%s_%s_seq', (SELECT max(%s) FROM \"%s\"));\n", dbi.QuoteEscape(tableName), dbi.QuoteEscape(column.ColumnName), column.ColumnName, tableName)
			if _, err := writer.Write([]byte(seq)); err != nil {
				return err
			}
		}
	}

	return nil
}
