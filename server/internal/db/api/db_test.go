package api

import (
	"github.com/stretchr/testify/require"
	"mayfly-go/internal/db/domain/entity"
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
