package application

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProposalAckWireValues 审批回执的线协议字面值必须钉住。
//
// 这些值被 flow 落进 biz_handle_res、再由前端审批面板按字符串识别（见 frontend
// views/flow/flowbiz/agentops/proposalAck.ts）。本断言不是形式主义：值一改，存量工单的回执
// 就再也认不出来，面板会从「等待提示」退化成裸直出 JSON，而两侧都不会报错。
// 改动这里的任何值，必须同步前端清单 + 存量数据说明。
func TestProposalAckWireValues(t *testing.T) {
	require.Equal(t, "proposalId", AckKeyProposalId)
	require.Equal(t, "status", AckKeyStatus)
	require.Equal(t, "execution", AckKeyExecution)
	require.Equal(t, "message", AckKeyMessage)

	require.Equal(t, "queued", AckExecutionQueued)
	require.Equal(t, "requeued", AckExecutionRequeued)
	require.Equal(t, "already_processed", AckMessageAlreadyProcessed)
	require.Equal(t, "already_claimed", AckMessageAlreadyClaimed)
}

// TestProposalAckValuesAreNotSentences 回执值必须是机器可读标识符，不能是待翻译的自然语言句子。
//
// 旧形态 `already processed` / `already claimed` 带空格、读起来像文案，下一个维护者会本能地
// 把它搬进 i18n 字典——而那会直接打断跨语言契约（前端按字面匹配，翻译后即失配）。
// 这条断言把「别把它当文案」变成机制而不靠人记。
func TestProposalAckValuesAreNotSentences(t *testing.T) {
	for _, v := range []string{AckExecutionQueued, AckExecutionRequeued, AckMessageAlreadyProcessed, AckMessageAlreadyClaimed} {
		require.NotContains(t, v, " ", "回执值 %q 含空格，形似待翻译文案", v)
	}
}

// TestProposalAckKeysCoverBothDiscriminators 受理回执的两个判别位键名必须互不重叠。
//
// 前端把「带 execution 或带 message」当作「这是一条无结论回执」的唯一判据；两个键若同名，
// 一种形态会覆盖另一种，面板把已投递执行的回执误判成幂等落败（或反之）
func TestProposalAckKeysCoverBothDiscriminators(t *testing.T) {
	require.NotEqual(t, AckKeyExecution, AckKeyMessage)
	require.NotEqual(t, AckExecutionQueued, AckExecutionRequeued)
	require.NotEqual(t, AckMessageAlreadyProcessed, AckMessageAlreadyClaimed)
}
