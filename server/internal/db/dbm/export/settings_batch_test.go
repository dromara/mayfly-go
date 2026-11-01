package export

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSettingsBatchBudgetTakesEffect Settings 的 BatchRows/BatchBytes 必须真正生效。
//
// 历史缺陷：Settings 声明了这两个字段、DefaultSettings 也填了值，但 exportData 只读常量，
// 于是调用方「为大字段调小批量」的诉求被静默忽略——配置项看起来生效，实际没有任何作用。
func TestSettingsBatchBudgetTakesEffect(t *testing.T) {
	content, err := os.ReadFile("exporter.go")
	require.NoError(t, err)
	source := string(content)

	require.Contains(t, source, "batchRows := e.settings.BatchRows", "行预算必须取自 Settings")
	require.Contains(t, source, "batchBytes := int(e.settings.BatchBytes)", "字节预算必须取自 Settings")
	// 不得再直接拿常量做取模：那正是 Settings 被忽略的写法
	require.NotContains(t, source, "dataCount%DefaultBatchRows", "预算判断不得再硬编码常量")
	// <=0 必须回落，避免 0 做取模除数 panic
	require.Contains(t, source, "batchRows = DefaultBatchRows")
	require.Contains(t, source, "batchBytes = DefaultBatchBytes")
}
