package entity

import (
	"testing"

	"mayfly-go/pkg/utils/stringx"

	"github.com/stretchr/testify/require"
)

// TestDiagnosisNotifyTemplateCoversAllParams 内置诊断通知模板必须覆盖 notifyResult 下发的每个参数键，
// 且用同源参数渲染后不得残留 <no value>。
//
// 这是「触发器/任务结果通知选了不匹配模板 → 内容全是 <no value>」的回归防线：新增或改名的参数键若
// 没同步进模板，或模板用了 params 里没有的占位符，这里立刻红（而不是等真机发出通知才发现）
func TestDiagnosisNotifyTemplateCoversAllParams(t *testing.T) {
	for _, key := range DiagNotifyParamKeys {
		require.Contains(t, DiagnosisNotifyTmpl, "{{."+key+"}}", "模板缺少参数占位符 %s", key)
	}

	// error 非空：覆盖 if 分支，失败原因应渲染出来
	params := NewDiagnosisNotifyParams("机器P0告警诊断", "run-1", "event", "succeeded", "结论摘要", "boom", 123)
	rendered, err := stringx.TemplateResolve(DiagnosisNotifyTmpl, params)
	require.NoError(t, err)
	require.NotContains(t, rendered, "<no value>", "渲染结果不得含未替换占位符")
	require.Contains(t, rendered, "机器P0告警诊断")
	require.Contains(t, rendered, "失败原因：boom")

	// error 为空：不应出现「失败原因」行（if 分支为假）
	paramsOK := NewDiagnosisNotifyParams("t", "r", "cron", "succeeded", "ok", "", 1)
	renderedOK, err := stringx.TemplateResolve(DiagnosisNotifyTmpl, paramsOK)
	require.NoError(t, err)
	require.NotContains(t, renderedOK, "失败原因")
	require.NotContains(t, renderedOK, "<no value>")
	require.Contains(t, renderedOK, "消耗Tokens：1")
}
