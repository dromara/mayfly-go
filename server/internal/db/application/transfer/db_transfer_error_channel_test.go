package transfer

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunDistinguishesStrategyErrorFromUnsupportedMode 策略自身失败不得被收尾成"不支持的模式"，
// 也不得重复收尾。
//
// 历史形态：Run 对 Pipeline.Execute 的任何 error 都调
// EndTransfer(..., "unsupported transfer mode", ...)。现有策略都是「void + 内部 EndTransfer」，
// 所以看起来没事；一旦某个策略按接口契约 return err，会先被内部收尾一次、再被这里收尾一次
// （终态写两遍、通知发两遍），且真实原因被覆盖成误导性的"不支持的模式"。
func TestRunDistinguishesStrategyErrorFromUnsupportedMode(t *testing.T) {
	content, err := os.ReadFile("db_transfer.go")
	require.NoError(t, err)
	source := string(content)

	require.Contains(t, source, "errors.As(err, &unsupported)", "必须按错误类型区分 unsupported 与策略失败")
	require.Contains(t, source, `"transfer failed"`, "策略失败要有独立文案，不能复用 unsupported")
	require.Contains(t, source, "log.Status == entity.DbTransferLogStatusRunning",
		"策略已自行收尾时不得重复收尾（仅当日志仍为 Running 才补）")

	// 不得再出现「任何 error 都当作 unsupported」的写法
	require.NotContains(t, strings.ReplaceAll(source, "\t", ""), "err != nil {\napp.EndTransfer(ctx, logId, taskId, \"unsupported transfer mode\", err, nil)")
}
