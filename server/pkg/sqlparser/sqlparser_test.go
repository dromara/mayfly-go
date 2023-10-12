package sqlparser

import (
	"strings"
	"testing"

	"github.com/kanzihuang/vitess/go/vt/sqlparser"
	"github.com/stretchr/testify/require"
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
			token := sqlparser.NewReaderTokenizer(strings.NewReader(test.input))
			tree, err := sqlparser.ParseNext(token)
			if len(test.err) > 0 {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.err)
				return
			}
			require.NoError(t, err)
			if len(test.want) == 0 {
				test.want = test.input
			}
			require.Equal(t, test.want, sqlparser.String(tree))
		})
	}
	// for _, test := range tests {
	// 	t.Run(test.name, func(t *testing.T) {
	// 		token := sqlparser_xwb1989.NewTokenizer(strings.NewReader(test.input))
	// 		tree, err := sqlparser_xwb1989.ParseNext(token)
	// 		if len(test.err) > 0 {
	// 			require.Error(t, err)
	// 			require.Contains(t, err.Error(), test.err)
	// 			return
	// 		}
	// 		require.NoError(t, err)
	// 		if len(test.want) == 0 {
	// 			test.want = test.input
	// 		}
	// 		require.Equal(t, test.wantXwb1989, sqlparser_xwb1989.String(tree))
	// 	})
	// }
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
			scanner := splitSqls(strings.NewReader(test.input))
			require.True(t, scanner.Scan())
			got := scanner.Text()
			if len(test.want) == 0 {
				test.want = test.input
			}
			require.Equal(t, test.want, got)
		})
	}
}
