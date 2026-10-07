package mongoexport

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/mongo/application/mongodoc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func queryDocs(t *testing.T, docs ...bson.D) []*mongodoc.QueryDoc {
	t.Helper()

	res := make([]*mongodoc.QueryDoc, 0, len(docs))
	for _, doc := range docs {
		encoded, err := mongodoc.EncodeQueryDoc(doc)
		require.NoError(t, err)
		res = append(res, encoded)
	}
	return res
}

func TestSupportedFormats(t *testing.T) {
	for _, format := range Formats {
		assert.True(t, Supported(format), "declared format %s must be supported", format)
	}
	assert.False(t, Supported("xlsx"))
	assert.False(t, Supported(""))
}

// TestExportJSONLinesKeepsTypes 导出的每一行必须仍是保真编码：
// 用普通 JSON 导出会把 Date/NumberLong 变成裸文本，再导入即完成类型损坏。
func TestExportJSONLinesKeepsTypes(t *testing.T) {
	docs := queryDocs(t,
		bson.D{{Key: "_id", Value: bson.NewObjectID()}, {Key: "at", Value: bson.NewDateTimeFromTime(time.Unix(1700000000, 0).UTC())}},
		bson.D{{Key: "_id", Value: int64(9007199254740993)}, {Key: "n", Value: int32(1)}},
	)

	content, contentType, err := Export(FormatJSON, docs)
	require.NoError(t, err)
	assert.Contains(t, contentType, "ndjson")

	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	require.Len(t, lines, 2, "行数必须与文档数一一对应")
	assert.Contains(t, lines[0], "$date")
	assert.NotContains(t, lines[0], "\n  ", "每行必须压缩为单行")
	assert.Contains(t, lines[1], "$numberLong")
	// _id 是 ObjectID 时享受展示豁免，导出为十六进制字符串；int64 主键则必须带类型包装
	assert.Contains(t, lines[1], `"_id":{"$numberLong":"9007199254740993"}`)
}

// TestExportCSVColumnUnion 表头取字段首次出现顺序的并集，缺字段的行留空。
func TestExportCSVColumnUnion(t *testing.T) {
	docs := queryDocs(t,
		bson.D{{Key: "_id", Value: "a-1"}, {Key: "name", Value: "first"}, {Key: "qty", Value: int32(2)}},
		bson.D{{Key: "_id", Value: "a-2"}, {Key: "note", Value: nil}, {Key: "name", Value: "second"}},
	)

	content, contentType, err := Export(FormatCSV, docs)
	require.NoError(t, err)
	assert.Contains(t, contentType, "text/csv")

	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, "_id,name,qty,note", lines[0], "列顺序必须是首次出现顺序，且第二行的新字段排在后面")
	assert.Equal(t, "a-1,first,2,", lines[1])
	assert.Equal(t, `a-2,second,,`, lines[2])
}

// TestExportCSVEscapesSeparatorsAndQuotes 分隔符与引号必须被转义，否则导出文件重新打开时行列会错位。
func TestExportCSVEscapesSeparatorsAndQuotes(t *testing.T) {
	docs := queryDocs(t, bson.D{
		{Key: "_id", Value: "x"},
		{Key: "text", Value: `has,comma and "quote"`},
		{Key: "multi", Value: "line1\nline2"},
	})

	content, err := ExportCSV(docs)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"has,comma and ""quote"""`)
	assert.Contains(t, string(content), "\"line1\nline2\"")

	// 表头 + 一行数据（该行内部含换行，所以总行数会多一行）：用解析回来验证才算严谨
	assert.True(t, strings.HasPrefix(string(content), "_id,text,multi\n"), "表头必须完整")
}

// TestExportCSVTypesAreReadable 单元格里的类型要能看出来，
// 否则导出后无法区分数字 1 与字符串 "1"，也没法判断日期是否被写坏。
func TestExportCSVTypesAreReadable(t *testing.T) {
	docs := queryDocs(t, bson.D{
		{Key: "_id", Value: bson.NewObjectID()},
		{Key: "at", Value: bson.NewDateTimeFromTime(time.Unix(1700000000, 0).UTC())},
		{Key: "amount", Value: mustDecimal(t, "129.005")},
		{Key: "items", Value: bson.A{int32(1), "x"}},
	})

	content, err := ExportCSV(docs)
	require.NoError(t, err)

	// 按 CSV 解析回来再断言：直接比文本会把引号转义误判成内容不符
	header, row := parseFirstRow(t, string(content))
	assert.Equal(t, []string{"_id", "at", "amount", "items"}, header)
	assert.Regexp(t, `^[0-9a-f]{24}$`, row[0], "ObjectID 导出为十六进制串")
	assert.Equal(t, "2023-11-14T22:13:20.000Z", row[1], "Date 必须输出可读时间而不是毫秒数或对象")
	assert.Equal(t, "129.005", row[2])
	assert.Equal(t, `[1,"x"]`, row[3], "数组以紧凑 JSON 进单元格")
}

// parseFirstRow 解析 CSV 的表头与首个数据行
func parseFirstRow(t *testing.T, content string) ([]string, []string) {
	t.Helper()

	records, err := csv.NewReader(strings.NewReader(content)).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2, "表头 + 一行数据")
	return records[0], records[1]
}

func TestExportRejectsUnknownFormat(t *testing.T) {
	_, _, err := Export("parquet", queryDocs(t, bson.D{{Key: "_id", Value: "x"}}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parquet")
}

func TestExportEmptyDocs(t *testing.T) {
	jsonContent, _, err := Export(FormatJSON, nil)
	require.NoError(t, err)
	assert.Empty(t, jsonContent)

	csvContent, _, err := Export(FormatCSV, []*mongodoc.QueryDoc{})
	require.NoError(t, err)
	assert.Empty(t, csvContent, "没有数据时不产出只有分隔符的表头")
}

func mustDecimal(t *testing.T, s string) bson.Decimal128 {
	t.Helper()
	d, err := bson.ParseDecimal128(s)
	require.NoError(t, err)
	return d
}
