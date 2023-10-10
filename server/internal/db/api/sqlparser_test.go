package api

import (
	"github.com/stretchr/testify/require"
	"github.com/xwb1989/sqlparser"
	"strings"
	"testing"
)

func Test_ParseNext_WithCurrentDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        string
		wantXwb1989 string
		err         string
	}{
		{
			name:  "create table with current_timestamp",
			input: "create table tbl (\n\tcreate_at datetime default current_timestamp()\n)",
			// xwb1989/sqlparser 不支持 current_timestamp()
			wantXwb1989: "create table tbl",
		},
		{
			name:  "create table with current_date",
			input: "create table tbl (\n\tcreate_at date default current_date()\n)",
			// xwb1989/sqlparser 不支持 current_date()
			wantXwb1989: "create table tbl",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := sqlparser.NewTokenizer(strings.NewReader(test.input))
			tree, err := sqlparser.ParseNext(token)
			if len(test.err) > 0 {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantXwb1989, sqlparser.String(tree))
		})
	}
}
