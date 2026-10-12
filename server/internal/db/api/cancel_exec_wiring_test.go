package api

// 取消执行入口的保码接线测试。
//
// 「刚刚结束，无需取消」带独立分流码 4003（sqlexec.CodeExecJustFinished）：前端中心层按码
// 压掉红色 toast——这是一个事实而非失败，结果区的状态迁移本身就是反馈。
// 而 biz.ErrIsNil 会把任何错误重包成 code=400：码一丢，前端判不出分流，
// 操作者就会被一个不是错误的事实喊话（与 redis mustExec / flowapp.PreserveDecisionCode 同一类失效）。
// 所以 handler 必须把登记表的业务错误原样 panic，本测试钉住这条接线。
//
// 运行：cd server && go test -count=1 -run TestCancelExec ./internal/db/api/

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelExecKeepsTheJustFinishedCode(t *testing.T) {
	source, err := os.ReadFile("db_session.go")
	require.NoError(t, err)

	body := string(source)
	start := strings.Index(body, "func (d *DbSession) CancelExec(")
	require.NotEqual(t, -1, start, "CancelExec handler 应存在")
	fn := body[start:]
	end := strings.Index(fn, "\n}")
	require.NotEqual(t, -1, end, "应能找到 CancelExec 的结束位置")
	fn = fn[:end]

	panicPos := strings.Index(fn, "panic(bizErr)")
	wrapPos := strings.Index(fn, "biz.ErrIsNil(err)")
	require.NotEqual(t, -1, panicPos, "业务错误必须原样 panic 保住分流码")
	require.NotEqual(t, -1, wrapPos, "非业务错误仍交回常规断言")
	assert.Less(t, panicPos, wrapPos, "保码必须发生在兜底重包之前")
}
