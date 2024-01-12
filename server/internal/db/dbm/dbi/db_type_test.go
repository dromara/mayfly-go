package dbi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_QuoteLiteral(t *testing.T) {
	tests := []struct {
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
			dbType: DbTypePostgres,
			sql:    "a\u00A0b",
			want:   "'a\u00A0b'",
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.dbType)+"_"+tt.sql, func(t *testing.T) {
			got := tt.dbType.QuoteLiteral(tt.sql)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_quoteIdentifier(t *testing.T) {
	tests := []struct {
		dbType DbType
		sql    string
		want   string
	}{
		{
			dbType: DbTypeMysql,
			sql:    "`a`",
		},
		{
			dbType: DbTypeMysql,
			sql:    "select table",
		},
		{
			dbType: DbTypePostgres,
			sql:    "a",
		},
		{
			dbType: DbTypePostgres,
			sql:    "table",
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.dbType)+"_"+tt.sql, func(t *testing.T) {
			got := tt.dbType.QuoteIdentifier(tt.sql)
			require.Equal(t, tt.want, got)
		})
	}
}
