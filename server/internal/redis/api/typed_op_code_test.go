package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTypedOpsPreserveDecisionCode 面板的写操作必须用保码断言取结果。
//
// 命令台第一版实测就栽在丢码上：biz.ErrIsNil 把 4002 重包成 400，前端 isWarnAckError 判假，
// 确认框不出现、只剩一句红色提示，而提示写的还是「确认后将直接执行」——界面上根本没有确认入口。
// 面板操作与命令台共用同一套确认流，所以每个写入口都得走 mustExec，漏一个就退回那个失效形态
func TestTypedOpsPreserveDecisionCode(t *testing.T) {
	content, err := os.ReadFile("key_value.go")
	require.NoError(t, err)
	source := string(content)

	// 7 = 成员写入、视角操作、设置 TTL、重命名、复制、删除 key，加上 key 详情的读内容
	require.Equal(t, 7, strings.Count(source, "\n\tmustExec("), "面板入口数量变化时同步此处")
	require.NotContains(t, source, "biz.ErrIsNil(r.keyValueApp.", "面板写操作不能直接用会丢码的断言")

	fn := source[strings.Index(source, "func mustExec(err error) {"):]
	fn = fn[:strings.Index(fn, "\n}")]
	require.Contains(t, fn, "flowapp.PreserveDecisionCode(err)", "保码必须发生在断言之前")
	require.Less(t, strings.Index(fn, "PreserveDecisionCode"), strings.Index(fn, "biz.ErrIsNil"))
}

// TestContentReadingIsCheckedBeforeLoading 面板读内容必须先过策略再看数据，且拦截要保码。
//
// 「申请查看」这个入口能否出现，取决于 4001/4002 有没有原样送到前端：
// 码一丢，前端只会弹一句红色报错，既弹不出确认框也不会给查看申请入口
func TestContentReadingIsCheckedBeforeLoading(t *testing.T) {
	appSrc, err := os.ReadFile(filepath.Join("..", "application", "key_value.go"))
	require.NoError(t, err)
	page := string(appSrc)[strings.Index(string(appSrc), "func (k *keyValueAppImpl) Page("):]
	page = page[:strings.Index(page, "\n}")]
	require.Greater(t, strings.Index(page, "checkReadTrigger"), 0, "读内容没过触发策略")
	require.Less(t, strings.Index(page, "checkReadTrigger"), strings.Index(page, "handler.Load"), "必须先判定再取数据")

	apiSrc, err := os.ReadFile("key_value.go")
	require.NoError(t, err)
	handler := string(apiSrc)[strings.Index(string(apiSrc), "func (r *Redis) KeyValues("):]
	handler = handler[:strings.Index(handler, "\n}")]
	require.Contains(t, handler, "mustExec(err)", "读内容的拦截必须保码，否则前端给不出「申请查看」")
	// 读内容给得出「申请查看」，属可提单入口：用 Direct 会让提示把用户支到命令控制台
	require.Contains(t, handler, "application.WarnAckOf(pageForm.AckWarn)")
}
