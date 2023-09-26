package api

import (
	"github.com/kanzihuang/vitess/go/vt/sqlparser"
	"github.com/stretchr/testify/require"
	sqlparser_xwb1989 "github.com/xwb1989/sqlparser"
	"strings"
	"testing"
)

func Test_ParseNext(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		want         string
		want_xwb1989 string
		err          string
	}{
		{
			name:         "create table with current_timestamp",
			input:        "create table tbl (id bigint(20), create_at datetime DEFAULT current_timestamp())",
			want:         "create table tbl (\n\tid bigint(20),\n\tcreate_at datetime default current_timestamp()\n)",
			want_xwb1989: "create table tbl", // xwb1989/sqlparser 不支持 current_timestamp()
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := sqlparser.NewReaderTokenizer(strings.NewReader(test.input))
			tree, err := sqlparser.ParseNext(token)
			if len(test.err) > 0 {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, sqlparser.String(tree))
		})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := sqlparser_xwb1989.NewTokenizer(strings.NewReader(test.input))
			tree, err := sqlparser_xwb1989.ParseNext(token)
			if len(test.err) > 0 {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want_xwb1989, sqlparser_xwb1989.String(tree))
		})
	}
}
