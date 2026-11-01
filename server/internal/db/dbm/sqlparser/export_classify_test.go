package sqlparser_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

var updateClassifyCases = flag.Bool("update-classify", false, "把复杂 SQL 准确性语料导出到 testdata/classify_cases.json")

type classifyCaseJSON struct {
	Name     string `json:"name"`
	DBType   string `json:"dbType"`
	SQL      string `json:"sql"`
	StmtType string `json:"stmtType"`
	IsQuery  bool   `json:"isQuery"`
}

// TestExportClassifyCases 把 TestComplexSQLAccuracy 的语料导出成跨语言契约。
//
// 目的：同一批语料同时驱动「后端 Go 断言」与「前端 TS 的 isQueryStatement 断言」，
// 任一侧漂移都会立刻变红（避免再出现"后端只读、前端写入"这类分叉）。
// 用法：go test ./internal/db/dbm/sqlparser/ -run TestExportClassifyCases -update-classify
func TestExportClassifyCases(t *testing.T) {
	if !*updateClassifyCases {
		t.Skip("use -update-classify to regenerate testdata/classify_cases.json")
	}
	const path = "testdata/classify_cases.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var doc struct {
		Comment json.RawMessage    `json:"_comment"`
		Cases   []classifyCaseJSON `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse contract: %v", err)
	}

	// 幂等：先移除上一次导出的 acc-* 条目，再按当前语料重写（否则规则调整后旧值不会更新）
	kept := make([]classifyCaseJSON, 0, len(doc.Cases))
	for _, c := range doc.Cases {
		if strings.HasPrefix(c.Name, "acc-") {
			continue
		}
		kept = append(kept, c)
	}
	doc.Cases = kept

	seen := make(map[string]bool, len(doc.Cases))
	for _, c := range doc.Cases {
		seen[c.DBType+"|"+c.SQL] = true
	}

	sets := []struct {
		dbType string
		cases  []accuracyCase
	}{
		{"mysql", append(append([]accuracyCase{}, accMy...), accCommon...)},
		{"postgres", append(append([]accuracyCase{}, accPg...), accCommon...)},
		{"oracle", append(append([]accuracyCase{}, accOr...), accCommon...)},
		{"dm", append(append([]accuracyCase{}, accOr...), accCommon...)},
		{"mssql", append(append([]accuracyCase{}, accMs...), accCommon...)},
		{"sqlite", append(append([]accuracyCase{}, accSl...), accCommon...)},
		{"clickhouse", append(append([]accuracyCase{}, accCh...), accCommon...)},
	}
	added := 0
	for _, set := range sets {
		for i, c := range set.cases {
			key := set.dbType + "|" + c.sql
			if seen[key] {
				continue
			}
			seen[key] = true
			doc.Cases = append(doc.Cases, classifyCaseJSON{
				Name:     fmt.Sprintf("acc-%s-%d", set.dbType, i),
				DBType:   set.dbType,
				SQL:      c.sql,
				StmtType: string(c.wantType),
				IsQuery:  isQueryExpected(c.sql, c.wantType),
			})
			added++
		}
	}

	out, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Logf("exported %d new cases (total %d)", added, len(doc.Cases))
}

// isQueryExpected 推演前端 isQueryStatement 的期望值（前端只读集合 + 隐藏写降级）。
//
// 前端规则：首词 ∈ {select,with,table,show,explain,describe,desc} 且体内代码区无隐藏写。
// 因此 other-read 里只有 show/desc/explain（非 ANALYZE）算查询；SET/VACUUM/PRAGMA 不算。
func isQueryExpected(sql string, typ sqlstmt.StmtType) bool {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	fields := strings.Fields(upper)
	if len(fields) == 0 {
		return false
	}
	first := fields[0]
	switch typ {
	case sqlstmt.StmtTypeSelect:
		switch first {
		case "WITH":
			// 数据修改型 CTE（体内 DELETE/UPDATE/INSERT/MERGE）
			return !containsAnyWord(upper, "DELETE", "UPDATE", "INSERT", "MERGE")
		case "SELECT":
			// SELECT ... INTO 建表/导出，是写
			return !strings.Contains(upper, " INTO ")
		}
		return true
	case sqlstmt.StmtTypeOtherRead:
		switch first {
		case "SHOW", "DESCRIBE", "DESC", "TABLE", "PRAGMA":
			// PRAGMA 是 sqlite 的只读扩展词（前端 dialect 能力位 queryStatementKeywords 含它）
			return true
		case "EXPLAIN":
			// EXPLAIN ANALYZE 会真执行内层语句
			return !strings.Contains(upper, "ANALYZE") && !strings.Contains(upper, "ANALYSE")
		}
		return false
	}
	return false
}

func containsAnyWord(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
