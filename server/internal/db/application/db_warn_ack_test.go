package application

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDataImportIsGoverned 数据导入必须与 SQL 编辑器过同一套触发策略。
//
// 两者用的是同一个权限码（db:sqlscript:run），平台已把它归为同一能力；此前导入直接
// TxExecContext 写库、一次策略都不查：编辑器里「insert 需审批」拦得住，换成导入文件就绕过。
//
// 本用例留在父包：它守的是留守文件 db_data_import.go 的接线（经 GetDbSQLExecApp() 取闸门），
// 与 sqlexec 内部实现无关，故按源码相对路径就地扫描。
func TestDataImportIsGoverned(t *testing.T) {
	content, err := os.ReadFile("db_data_import.go")
	require.NoError(t, err)
	source := string(content)

	judge := strings.Index(source, "GetDbSQLExecApp().CheckSqlsWithoutTicket(ctx, conn, probeStmts)")
	begin := strings.Index(source, "tx, err := conn.Begin()")
	require.Greater(t, judge, -1, "数据导入没过触发策略")
	require.Less(t, judge, begin, "必须在开事务前判定，不能写完再问")

	// 判定用「插入一行」的代表语句：批次大小是实现细节，不该决定策略结论
	require.Contains(t, source, "values[:1]", "判定语句必须与批次大小无关")
	// 提醒不阻断，但要回传给操作者
	require.Contains(t, source, "Warnings: warnings")
}
