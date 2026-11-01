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

	// 表/列访问控制同样必须早于写库：导入是「写」入口，只过触发策略等于给保护名单开了后门
	// 列级必须带上映射列：AssertColumnsAllowed 对空 refs 直接放行，
	// 传 nil 等于只判表级、列级保护被整条绕过（可覆盖敏感列取值）。
	// 列引用由 buildImportColumnRefs 统一构造（传 nil 等于只判表级，列保护会被绕过）
	aclJudge := strings.Index(source, "d.tableAcl.AssertStatementAllowed(ctx, conn.Info.Id, []string{req.TableName}, true, buildImportColumnRefs(")
	require.Greater(t, aclJudge, -1, "数据导入没过表/列访问控制")
	require.Less(t, aclJudge, begin, "表/列访问控制必须在开事务前判定")
	require.Contains(t, source, "sqlexec.ColumnRef{Table: tableName, Column: column.ColumnName}", "列级访问控制必须按映射列判定，不能传 nil")
}
