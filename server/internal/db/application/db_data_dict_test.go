package application

// 数据字典的文本渲染：交付物必须能被 Excel / Markdown 稳定解析，
// 逗号、引号、换行、竖线这几个字符是唯一会破坏结构的输入。

import (
	"strings"
	"testing"
)

func sampleDict() *DataDict {
	return &DataDict{
		Db: "mayfly_go",
		Tables: []DictTable{
			{Name: "t_a", Comment: "第一张表, 含逗号", Rows: 10, Bytes: 2048},
			{Name: "t_b", Comment: "", Rows: 0, Bytes: 0},
		},
		Columns: []DictColumn{
			{Table: "t_a", Name: "id", Type: "int", Comment: "主键|带竖线", PrimaryKey: true},
			{Table: "t_a", Name: "name", Type: "varchar(64)", Comment: "含\"引号\"\n换行", Nullable: true},
			{Table: "t_b", Name: "payload", Type: "json", Generated: true},
		},
		Indexes: []DictIndex{
			{Table: "t_a", Name: "PRIMARY", Columns: "id", Unique: true, Primary: true},
		},
	}
}

func TestRenderCSV(t *testing.T) {
	output := string((&dataDictAppImpl{}).RenderCSV(sampleDict()))

	// Excel 打开中文不乱码靠 BOM
	if !strings.HasPrefix(output, "\ufeff") {
		t.Fatal("csv must start with BOM")
	}
	lines := strings.Split(strings.TrimRight(output, "\r\n"), "\r\n")
	if len(lines) != 4 {
		t.Fatalf("want header + 3 column rows, got %d lines: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "表名") || !strings.Contains(lines[0], "列名") {
		t.Fatalf("header must carry Chinese column names, got %q", lines[0])
	}
	// 逗号与引号必须被转义成带引号字段，否则列会错位
	if !strings.Contains(output, `"第一张表, 含逗号"`) {
		t.Fatalf("comma in comment must be quoted, got %s", output)
	}
	if !strings.Contains(output, `"含""引号""`) {
		t.Fatalf("double quote must be doubled, got %s", output)
	}
	// 生成列要如实标出
	if !strings.Contains(output, ",true,") && !strings.Contains(output, "json") {
		t.Fatalf("generated column must be exported, got %s", output)
	}
}

func TestRenderMarkdown(t *testing.T) {
	output := string((&dataDictAppImpl{}).RenderMarkdown(sampleDict()))

	if !strings.Contains(output, "# 数据字典：mayfly_go") {
		t.Fatalf("markdown must carry title, got %s", output)
	}
	if !strings.Contains(output, "共 2 张表、3 个列、1 个索引") {
		t.Fatalf("markdown must summarize counts, got %s", output)
	}
	// 一表一节
	if strings.Count(output, "\n## ") < 2 {
		t.Fatalf("each table must have its own section, got %s", output)
	}
	// 竖线转义：否则表格列会错位
	if strings.Contains(output, "主键|带竖线") {
		t.Fatalf("pipe must be escaped in table cells, got %s", output)
	}
	if !strings.Contains(output, `主键\|带竖线`) {
		t.Fatalf("escaped pipe expected, got %s", output)
	}
	// 换行压成空格，保证一行一条记录
	if strings.Contains(output, "换行\n") {
		t.Fatalf("newline in comment must be flattened, got %s", output)
	}
}
