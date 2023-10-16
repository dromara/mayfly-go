package api

import (
	"github.com/stretchr/testify/require"
	"mayfly-go/internal/db/domain/entity"
	"testing"
)

func Test_escapeSql(t *testing.T) {
	tests := []struct {
		name   string
		dbType entity.DBType
		sql    string
		want   string
	}{
		{
			dbType: entity.DBTypeMysql{},
			sql:    "\\a\\b",
			want:   "'\\\\a\\\\b'",
		},
		{
			dbType: entity.DBTypeMysql{},
			sql:    "'a'",
			want:   "'''a'''",
		},
		{
			name:   "不间断空格",
			dbType: entity.DBTypeMysql{},
			sql:    "a\u00A0b",
			want:   "'a\u00A0b'",
		},
		{
			dbType: entity.DBTypePostgres{},
			sql:    "\\a\\b",
			want:   " E'\\\\a\\\\b'",
		},
		{
			dbType: entity.DBTypePostgres{},
			sql:    "'a'",
			want:   "'''a'''",
		},
		{
			name:   "不间断空格",
			dbType: entity.DBTypePostgres{},
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
