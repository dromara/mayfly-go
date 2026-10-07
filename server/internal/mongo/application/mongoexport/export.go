// Package mongoexport 把查询出的文档写成可下载文件。
//
// 只做纯转换（不持有连接、不写 HTTP），因此格式行为可以完全单测；
// 行数上限、权限与审计都由调用方在取数与响应阶段负责。
package mongoexport

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"

	"mayfly-go/internal/mongo/application/mongodoc"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// 导出格式。取值与 API 的 format 参数一致，前端下拉也用它，避免两处各写一份字面量。
const (
	FormatJSON = "json"
	FormatCSV  = "csv"
)

// Formats 支持的导出格式（顺序即前端展示顺序）。
var Formats = []string{FormatJSON, FormatCSV}

// Supported 报告格式是否受支持。
func Supported(format string) bool {
	for _, item := range Formats {
		if item == format {
			return true
		}
	}
	return false
}

// Export 把文档转成文件内容。
func Export(format string, docs []*mongodoc.QueryDoc) ([]byte, string, error) {
	switch format {
	case FormatJSON:
		b, err := ExportJSONLines(docs)
		return b, "application/x-ndjson; charset=utf-8", err
	case FormatCSV:
		b, err := ExportCSV(docs)
		return b, "text/csv; charset=utf-8", err
	default:
		return nil, "", fmt.Errorf("mongoexport: unsupported format %q", format)
	}
}

// ExportJSONLines 每行一个文档，内容是服务端的保真编码（含 $date/$oid 等类型包装）。
//
// 用换行分隔的 JSON 而不是一个大数组：几万条时人可以流式查看与 grep，
// 且单条文档的形状与服务端返回、与写回时的入参完全一致。
func ExportJSONLines(docs []*mongodoc.QueryDoc) ([]byte, error) {
	buf := &bytes.Buffer{}
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		// 压缩成单行：缩进版会让行数与文档数不再一一对应
		compact := &bytes.Buffer{}
		if err := json.Compact(compact, doc.Doc); err != nil {
			return nil, err
		}
		buf.Write(compact.Bytes())
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// ExportCSV 以「字段首次出现顺序」的并集作为表头。
//
// Mongo 文档是 schema-less 的，列只能从本次导出的数据里推；缺字段的行留空，
// 嵌套文档与数组以紧凑 JSON 原样进单元格（拆开会让列名与原文对不上，回查困难）。
func ExportCSV(docs []*mongodoc.QueryDoc) ([]byte, error) {
	header := make([]string, 0, 16)
	columnIndex := make(map[string]int)
	rows := make([][]cell, 0, len(docs))

	for _, doc := range docs {
		if doc == nil {
			continue
		}
		decoded, err := mongodoc.Decode(doc.Doc)
		if err != nil {
			return nil, err
		}

		row := make([]cell, 0, len(decoded))
		for _, e := range decoded {
			if _, ok := columnIndex[e.Key]; !ok {
				columnIndex[e.Key] = len(header)
				header = append(header, e.Key)
			}
			value, err := stringifyValue(e.Value)
			if err != nil {
				return nil, err
			}
			row = append(row, cell{key: e.Key, value: value})
		}
		rows = append(rows, row)
	}

	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if len(header) == 0 {
		// 没有数据时不写只含分隔符的空表头：那会让一个空集合导出成一行脏数据
		return nil, nil
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, row := range rows {
		record := make([]string, len(header))
		for _, item := range row {
			record[columnIndex[item.key]] = item.value
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// cell 是一个已命名的单元格值：带字段名才能安全地按表头列序写入。
type cell struct {
	key   string
	value string
}

// stringifyValue 把 BSON 值转成单元格文本：类型走可读形态，避免导出后无法判断 `1` 是 int 还是字符串。
func stringifyValue(v any) (string, error) {
	switch val := v.(type) {
	case nil:
		return "", nil
	case string:
		return val, nil
	case bool:
		return strconv.FormatBool(val), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64), nil
	case bson.DateTime:
		return val.Time().UTC().Format("2006-01-02T15:04:05.000Z"), nil
	case bson.ObjectID:
		return val.Hex(), nil
	case bson.Decimal128:
		return val.String(), nil
	case bson.Binary:
		return base64.StdEncoding.EncodeToString(val.Data), nil
	case bson.Regex:
		return "/" + val.Pattern + "/" + val.Options, nil
	case bson.Timestamp:
		return fmt.Sprintf("Timestamp(%d, %d)", val.T, val.I), nil
	case bson.D, bson.M, bson.A:
		// 嵌套结构以 relaxed Extended JSON 进单元格：CSV 是给人和表格软件读的，
		// 写成 {"$numberInt":"1"} 会把可读性抵掉；需要类型无损往返时请用 json 格式导出
		encoded, err := bson.MarshalExtJSON(bson.D{{Key: "v", Value: val}}, false, false)
		if err != nil {
			return "", err
		}
		var wrapper map[string]json.RawMessage
		if err = json.Unmarshal(encoded, &wrapper); err != nil {
			return "", err
		}
		return string(wrapper["v"]), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}
