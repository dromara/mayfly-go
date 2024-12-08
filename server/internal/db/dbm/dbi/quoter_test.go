package dbi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoteTo(t *testing.T) {
	var (
		quoter = Quoter{'[', ']', AlwaysReserve}
		kases  = []struct {
			expected string
			value    string
		}{
			{"[table]", "table"},
			{"[table]", "[table]"},
			{`[table].*`, `[table].*`},
			{"[schema].[table]", "schema.table"},
			{`["schema].[table"]`, `"schema.table"`},
			{"[schema].[table] AS [table]", "schema.table AS table"},
			{" [table]", " table"},
			{"  [table]", "  table"},
			{"[table] ", "table "},
			{"[table]  ", "table  "},
			{" [table] ", " table "},
			{"  [table]  ", "  table  "},
		}
	)

	for _, v := range kases {
		t.Run(v.value, func(t *testing.T) {
			buf := &strings.Builder{}
			err := quoter.QuoteTo(buf, v.value)
			assert.NoError(t, err)
			assert.EqualValues(t, v.expected, buf.String())
		})
	}
}

func TestReversedQuoteTo(t *testing.T) {
	var (
		quoter = Quoter{'[', ']', func(s string) bool {
			return s == "table"
		}}
		kases = []struct {
			expected string
			value    string
		}{
			{"[table]", "table"},
			{"[table].*", `[table].*`},
			{`"table"`, `"table"`},
			{"schema.[table]", "schema.table"},
			{"[schema].[table]", `[schema].table`},
			{"schema.[table]", `schema.[table]`},
			{"[schema].[table]", `[schema].[table]`},
			{`"schema.table"`, `"schema.table"`},
			{"schema.[table] AS table1", "schema.table AS table1"},
		}
	)

	for _, v := range kases {
		t.Run(v.value, func(t *testing.T) {
			buf := &strings.Builder{}
			quoter.QuoteTo(buf, v.value)
			assert.EqualValues(t, v.expected, buf.String())
		})
	}
}

func TestJoin(t *testing.T) {
	cols := []string{"f1", "f2", "f3"}
	quoter := Quoter{'[', ']', AlwaysReserve}

	assert.EqualValues(t, "[a],[b]", quoter.Join([]string{"a", " b"}, ","))

	assert.EqualValues(t, "[a].*,[b].[c]", quoter.Join([]string{"a.*", " b.c"}, ","))

	assert.EqualValues(t, "[b] [a]", quoter.Join([]string{"b a"}, ","))

	assert.EqualValues(t, "[f1], [f2], [f3]", quoter.Join(cols, ", "))

	quoter.IsReserved = AlwaysNoReserve
	assert.EqualValues(t, "f1, f2, f3", quoter.Join(cols, ", "))
}

func TestQuotes(t *testing.T) {
	cols := []string{"f1", "f2", "t3.f3", "t4.*"}
	quoter := Quoter{'[', ']', AlwaysReserve}

	quotedCols := quoter.Quotes(cols)
	assert.EqualValues(t, []string{"[f1]", "[f2]", "[t3].[f3]", "[t4].*"}, quotedCols)
}

func TestTrim(t *testing.T) {
	kases := map[string]string{
		"[table_name]":          "table_name",
		"[schema].[table_name]": "schema.table_name",
	}

	for src, dst := range kases {
		assert.EqualValues(t, src, DefaultQuoter.Trim(src))
		assert.EqualValues(t, dst, Quoter{'[', ']', AlwaysReserve}.Trim(src))
	}
}
