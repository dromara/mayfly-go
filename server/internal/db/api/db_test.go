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

func TestSplitSqls(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "select with '\\n'",
			input: "select 'hello\nworld';",
		},
		{
			name:  "create table `my-table`",
			input: "create table `my-table` (\n\t`my-id` bigint(20)\n)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.want) == 0 {
				tt.want = tt.input
			}
			scanner := SplitSqls(strings.NewReader(tt.input))
			require.True(t, scanner.Scan())
			require.Equal(t, tt.want, scanner.Text())
		})
	}
}

func Test_SplitSqls_WithLongString(t *testing.T) {
	const times = 0x1000
	sb := strings.Builder{}
	sb.WriteString("select '")
	for i := 0; i < times; i++ {
		sb.WriteString("\\n0123456789\n")
	}
	sb.WriteString("' from tbl")
	scanner := SplitSqls(strings.NewReader(sb.String()))
	require.True(t, scanner.Scan())
	require.Equal(t, sb.String(), scanner.Text())
}
