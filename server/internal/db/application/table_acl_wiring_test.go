package application

// 表级权限在「非 SQL 执行」入口的接线守卫。
//
// 这两个入口都不经过 SQL 编辑器，却是实打实的数据面：
//   - 值补全采样（ColumnValues）会 SELECT DISTINCT 具体列，把值捞回前端输入框；
//   - 复制表带数据时会 INSERT ... SELECT *，把源表整表搬进一张不受保护的新表。
// 漏接任意一处，保护名单就只是「编辑器里拦住」，换个按钮即可绕过。
//
// 运行：cd server && go test -count=1 -run TestTableAclWiring ./internal/db/application/

import (
	"os"
	"strings"
	"testing"
)

func TestTableAclWiring(t *testing.T) {
	columnValues, err := os.ReadFile("db_column_values.go")
	if err != nil {
		t.Fatalf("read db_column_values.go: %v", err)
	}
	cv := string(columnValues)
	assertPos := strings.Index(cv, "AssertTablesAllowed")
	queryPos := strings.Index(cv, "buildColumnValuesSQL")
	if assertPos == -1 {
		t.Fatal("值补全采样必须过表级权限")
	}
	if queryPos != -1 && assertPos > queryPos {
		t.Fatal("表级权限必须早于取数：拦不住等于没拦")
	}
	// 列级：采样的是「某一列的取值」，必须按列判，而不只是按表判
	if !strings.Contains(cv, "AssertColumnsAllowed") {
		t.Fatal("值补全采样必须过列级权限")
	}

	copyAPI, err := os.ReadFile("../api/db.go")
	if err != nil {
		t.Fatalf("read api/db.go: %v", err)
	}
	body := string(copyAPI)
	start := strings.Index(body, "func (d *Db) CopyTable(")
	if start == -1 {
		t.Fatal("CopyTable 应存在")
	}
	fn := body[start:]
	if end := strings.Index(fn[1:], "\nfunc "); end != -1 {
		fn = fn[:end+1]
	}
	guardPos := strings.Index(fn, "AssertWholeTableAllowed")
	copyPos := strings.Index(fn, "CopyTable(copy)")
	condPos := strings.Index(fn, "if copy.CopyData")
	if guardPos == -1 || condPos == -1 {
		t.Fatalf("复制表带数据时必须过整表可读判定（表级 + 列级）(guard=%d cond=%d)", guardPos, condPos)
	}
	if copyPos == -1 || guardPos > copyPos {
		t.Fatal("表级权限必须早于真正的复制动作")
	}
}
