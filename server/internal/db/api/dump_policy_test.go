package api

// 即席导出（dump）的治理接线测试。
//
// 导出是「大范围读取」：它读到的数据与 SQL 编辑器 SELECT 完全同源。此前 dump 只校验权限码，
// 一次策略都不查，于是出现「网格里查这张表被拦、导出同一张表一路绿灯」。
// 本测试钉住两件事：
//   - 导出必须过与执行 / 导入 / 查询导出同一道闸门（CheckSqlsWithoutTicket）；
//   - 判定必须在设置响应头之前——被拦时要返回 JSON 错误，而不是一个已写头、解压即坏的文件。
//
// 运行：cd server && go test -count=1 -run TestDumpIsGoverned ./internal/db/api/

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDumpIsGovernedBeforeWritingAnything(t *testing.T) {
	source, err := os.ReadFile("db.go")
	require.NoError(t, err)
	body := string(source)

	// 导出实现收口在 streamDump（即席导出与导出工单下载共用），
	// 因此「策略闸门 + 响应头」的顺序要在这一个函数里断言
	start := strings.Index(body, "func (d *Db) streamDump(")
	require.NotEqual(t, -1, start, "streamDump 应存在")
	fn := body[start:]
	// 取到下一个顶层函数为止，避免把后面 handler 的同名调用误算进本函数
	if next := strings.Index(fn[1:], "\nfunc "); next != -1 {
		fn = fn[:next+1]
	}

	// 工单通道可以跳过策略，但必须是显式开关，不能靠「忘了判」达成
	assert.Contains(t, fn, "SkipPolicy")

	checkPos := strings.Index(fn, "CheckSqlsWithoutTicket")
	headerPos := strings.Index(fn, `Content-Disposition`)
	selectsPos := strings.Index(fn, "sqlutil.ExportSelects")

	require.NotEqual(t, -1, checkPos, "dump 必须过触发策略闸门（CheckSqlsWithoutTicket）")
	require.NotEqual(t, -1, selectsPos, "导出判定应复用 sqlutil.ExportSelects 构造等价 SELECT，而不是另判一套")
	require.NotEqual(t, -1, headerPos, "导出响应头设置应存在")

	assert.Less(t, checkPos, headerPos, "策略判定必须早于写响应头：否则被拦时用户拿到的是半截文件")
	assert.Less(t, selectsPos, checkPos, "先构造判定语句再送判定")
}
