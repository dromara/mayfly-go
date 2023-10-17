package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_escapeSql(t *testing.T) {
	tests := []struct {
		name   string
		dbType DbType
		sql    string
		want   string
	}{
		{
			dbType: DbTypeMysql,
			sql:    "\\a\\b",
			want:   "'\\\\a\\\\b'",
		},
		{
			dbType: DbTypeMysql,
			sql:    "'a'",
			want:   "'''a'''",
		},
		{
			name:   "不间断空格",
			dbType: DbTypeMysql,
			sql:    "a\u00A0b",
			want:   "'a\u00A0b'",
		},
		{
			dbType: DbTypePostgres,
			sql:    "\\a\\b",
			want:   " E'\\\\a\\\\b'",
		},
		{
			dbType: DbTypePostgres,
			sql:    "'a'",
			want:   "'''a'''",
		},
		{
			name:   "不间断空格",
			dbType: DbTypePostgres,
			sql:    "a\u00A0b",
			want:   "'a\u00A0b'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dbType.QuoteLiteral(tt.sql)
			require.Equal(t, tt.want, got)
		})
	}
}
