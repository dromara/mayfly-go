package application

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWarnAckIsDeclaredByCaller 「是否追问仅提醒」必须由调用入口声明，不能靠语句条数推断。
//
// 曾经写成 requireWarnAck := len(sqlList) == 1，思路是「批量执行做不到逐条弹确认」。
// 但 SQL 控制台的批量执行在客户端本来就是一条一条发请求的，服务端看到的永远是单条：
// 于是批量选区里每条语句都被追问，而前端批量分支处理不了确认码，
// 最终表现是提示写着「可直接执行」却没有任何执行入口、一条语句也没跑
func TestWarnAckIsDeclaredByCaller(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(".", "db_sql_exec.go"))
	require.NoError(t, err)
	source := string(content)

	require.NotContains(t, source, "len(sqlList) == 1", "追问与否不能再靠语句条数推断")

	matched := regexp.MustCompile(`(?s)sqlExec := &sqlExecParam\{.*?\n\t\t\}`).FindString(source)
	require.NotEmpty(t, matched, "语句执行参数的构造结构变了，请同步检查提醒确认相关字段是否还在")
	require.Regexp(t, `RequireWarnAck:\s+execSQLReq\.RequireWarnAck`, matched)
	require.Regexp(t, `WarnAcknowledged:\s+execSQLReq\.WarnAcknowledged`, matched)
}

// TestOnlyConsoleEntryAsksForAck 只有能弹确认的入口才要求确认，也只有回放入口能跳过策略判定。
//
// 文件导入、审批回放这类入口拿不到操作者回应：若它们也要求确认，工单批完了还会
// 卡在一个没人在看的确认上。跳过策略判定也不得做成布尔位（零值即「不查」，新调用点
// 忘了置位就静默绕过治理），必须是显式点名的另一个入口
func TestOnlyConsoleEntryAsksForAck(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "api", "db.go"))
	require.NoError(t, err)
	require.Regexp(t, `RequireWarnAck:\s+form\.AskWarn`, string(content), "控制台入口要把前端的「能否弹确认」透传下来")

	execContent, err := os.ReadFile(filepath.Join(".", "db_sql_exec.go"))
	require.NoError(t, err)
	source := string(execContent)

	// 回放只能走 ExecApproved，且不声明任何确认位
	replay := regexp.MustCompile(`(?s)d\.ExecApproved\(ctx, &dto\.DbSQLExecReq\{.*?\n\t\}\)`).FindString(source)
	require.NotEmpty(t, replay, "回放调用结构变了，请同步确认它是否会被提醒确认卡住")
	require.NotContains(t, replay, "RequireWarnAck", "回放路径不能要求确认：没有人在那一步点确认框")

	// 用户发起的 Exec 必须无条件解析策略，且不能再有「要不要查」开关
	userEntry := source[strings.Index(source, "func (d *dbSQLExecAppImpl) Exec("):]
	userEntry = userEntry[:strings.Index(userEntry, "\n}")]
	require.Contains(t, userEntry, "GetProcdefByCodePath", "Exec 必须自己解析流程定义")
	require.NotContains(t, source, "CheckFlow", "不得再用布尔开关控制是否查策略")
	// 除回放外不得有人给内部 exec 传 nil 流程定义（那等于自己关掉治理）
	require.Equal(t, 1, strings.Count(source, "d.exec(ctx, execSQLReq, nil)"), "只有回放可以不带流程定义")
}

// TestImportedSqlGoesThroughTriggerCheck 文件导入必须逐条过触发策略。
//
// 导入循环原本是 dbConn.TxExec 直连执行，一次策略都不查：把「禁止执行」的语句写进 .sql 上传即可绕过，
// 比界面上少弹一个确认框严重得多。这里守住两件事：判定发生在执行之前，且导入不逐条追问
func TestImportedSqlGoesThroughTriggerCheck(t *testing.T) {
	content, err := os.ReadFile("db_sql_exec.go")
	require.NoError(t, err)
	source := string(content)

	start := strings.Index(source, "func (d *dbSQLExecAppImpl) ExecReader")
	require.Greater(t, start, -1, "找不到导入实现")
	body := source[start:]
	// 匹配完整形态：只写 _ = d.checkImportedSQL(...) 把结论丢掉同样等于不过策略
	check := strings.Index(body, "if err := d.checkImportedSQL(ctx, importProcdef, dbConn, sql); err != nil {")
	exec := strings.Index(body, "dbConn.TxExec(tx, sql)")
	require.Greater(t, check, -1, "导入循环里没有过策略判定")
	require.Less(t, check, exec, "必须先判定再执行，否则拦下的语句已经落库")

	// 导入过程里没有能逐条应答确认的人：判定入参不能带确认位，否则一个文件几百条语句会把导入变成人工批处理
	fnStart := strings.Index(source, "func (d *dbSQLExecAppImpl) checkImportedSQL")
	require.Greater(t, fnStart, -1, "找不到导入判定实现")
	fnBody := source[fnStart : strings.Index(source[fnStart:], "\n}")+fnStart]
	require.NotContains(t, fnBody, "RequireWarnAck", "导入不能声明能弹确认")
	require.NotContains(t, fnBody, "WarnAcknowledged", "导入没有操作者确认一说")

	require.Contains(t, body, "GetProcdefByCodePath", "流程定义按连接解析一次，不能逐条查库")
}

// TestCheckImportedSQLSkipsWithoutProcdef 未绑定流程定义的库不付解析开销
func TestCheckImportedSQLSkipsWithoutProcdef(t *testing.T) {
	app := &dbSQLExecAppImpl{}
	require.NoError(t, app.checkImportedSQL(context.Background(), nil, nil, "DROP TABLE whatever"))
}

// TestDataImportIsGoverned 数据导入必须与 SQL 编辑器过同一套触发策略。
//
// 两者用的是同一个权限码（db:sqlscript:run），平台已把它归为同一能力；此前导入直接
// TxExecContext 写库、一次策略都不查：编辑器里「insert 需审批」拦得住，换成导入文件就绕过
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
