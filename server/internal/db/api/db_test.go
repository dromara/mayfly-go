package api

import (
	"github.com/stretchr/testify/require"
	"mayfly-go/internal/db/domain/entity"
	"strings"
	"testing"
)

func Test_escapeSql(t *testing.T) {
	tests := []struct {
		name   string
		dbType string
		sql    string
		want   string
	}{
		{
			dbType: entity.DbTypeMysql,
			sql:    "\\a\\b",
			want:   "'\\\\a\\\\b'",
		},
		{
			dbType: entity.DbTypeMysql,
			sql:    "'a'",
			want:   "'''a'''",
		},
		{
			name:   "不间断空格",
			dbType: entity.DbTypeMysql,
			sql:    "a\u00A0b",
			want:   "'a\u00A0b'",
		},
		{
			dbType: entity.DbTypePostgres,
			sql:    "\\a\\b",
			want:   " E'\\\\a\\\\b'",
		},
		{
			dbType: entity.DbTypePostgres,
			sql:    "'a'",
			want:   "'''a'''",
		},
		{
			name:   "不间断空格",
			dbType: entity.DbTypePostgres,
			sql:    "a\u00A0b",
			want:   "'a\u00A0b'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeSql(tt.dbType, tt.sql)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_SplitSqls(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "create table with current_timestamp",
			input: "create table tbl (\n\tcreate_at datetime default current_timestamp()\n)",
		},
		{
			name:  "create table with current_date",
			input: "create table tbl (\n\tcreate_at date default current_date()\n)",
		},
		{
			name:  "select with ';\n'",
			input: "select 'the first line;\nthe second line;\n'",
			// SplitSqls split statements by ';\n'
			want: "select 'the first line;\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scanner := SplitSqls(strings.NewReader(test.input))
			require.True(t, scanner.Scan())
			got := scanner.Text()
			if len(test.want) == 0 {
				test.want = test.input
			}
			require.Equal(t, test.want, got)
		})
	}
}
