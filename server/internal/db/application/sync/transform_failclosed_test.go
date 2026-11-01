package sync

import (
	"os"
	"testing"

	"mayfly-go/internal/db/domain/entity"

	"github.com/stretchr/testify/require"
)

// TestTransformFailureIsFailClosed 转换规则失败必须留错并跳过该行，不能把 NULL 写进目标列。
//
// 历史行为：表达式求值失败/源列不存在时 `return nil, false` —— 值变成 NULL 且不计 skip、
// 不进运行日志，任务终态 Success。表现为"目标列从有值静默变成 NULL"，比直接报错难发现得多。
func TestTransformFailureIsFailClosed(t *testing.T) {
	fieldMap := []map[string]string{
		{"src": "id", "target": "id"},
		{"src": "v", "target": "v"},
	}
	task := &entity.DataSyncTask{}

	t.Run("源列不存在", func(t *testing.T) {
		engine, err := NewTransformEngine(`[{"targetColumn":"v","type":"column","config":{"sourceColumn":"missing_col"}}]`)
		require.NoError(t, err)
		require.NotNil(t, engine)
		data, skip := engine.Transform(map[string]any{"id": 1, "v": "original"}, fieldMap, task)
		require.Error(t, engine.Err(), "源列缺失必须留错")
		require.True(t, skip, "失败行应跳过，而不是把 NULL 写进目标列")
		require.Nil(t, data)
	})

	t.Run("表达式求值失败", func(t *testing.T) {
		engine, err := NewTransformEngine(`[{"targetColumn":"v","type":"expr","config":{"expression":"this is not an expression("}}]`)
		require.NoError(t, err)
		require.NotNil(t, engine)
		_, skip := engine.Transform(map[string]any{"id": 1, "v": "original"}, fieldMap, task)
		require.Error(t, engine.Err(), "表达式失败必须留错")
		require.True(t, skip)
	})

	t.Run("正常转换不留错", func(t *testing.T) {
		engine, err := NewTransformEngine(`[{"targetColumn":"v","type":"constant","config":{"value":"fixed"}}]`)
		require.NoError(t, err)
		require.NotNil(t, engine)
		data, skip := engine.Transform(map[string]any{"id": 1, "v": "original"}, fieldMap, task)
		require.NoError(t, engine.Err())
		require.False(t, skip)
		require.Equal(t, "fixed", data["v"])
	})
}

// TestStopTaskUsesStopSignal 停止任务必须走 RequestStop（写停止标记），而不是 Release。
//
// Release 在本地未持锁时直接返回：跨实例停止完全无效（执行侧一路写到底，界面显示已停止）；
// 本实例持锁时又会在协程仍写库的窗口内删锁，允许立刻再起一次（同表双写）。
func TestStopTaskUsesStopSignal(t *testing.T) {
	content, err := os.ReadFile("db_data_sync_runner.go")
	require.NoError(t, err)
	source := string(content)

	require.Contains(t, source, "app.runGuard.RequestStop(taskId)", "StopTask 必须写停止标记，否则跨实例停止无效")
	require.NotContains(t, source, "app.runGuard.Release(taskId)", "StopTask 不得直接删锁（会允许并发双写）")
}
