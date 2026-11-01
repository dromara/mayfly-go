package sync

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFullRefreshProbesSourceBeforeTruncate 全量刷新必须先探测源可读，再清空目标表。
//
// TRUNCATE 是 DDL（MySQL/PG 下不可回滚）。若先清空、再读源，源侧断连/权限被回收/
// SQL 运行期报错会留下"目标表全空 + 任务失败"的不可逆状态。
func TestFullRefreshProbesSourceBeforeTruncate(t *testing.T) {
	content, err := os.ReadFile("db_data_sync_exec.go")
	require.NoError(t, err)
	source := string(content)

	probeIdx := strings.Index(source, "target table was NOT cleared")
	truncateIdx := strings.Index(source, "GenTruncate(task.TargetTableName)")
	require.Greater(t, probeIdx, -1, "全量刷新缺少源可读性探针")
	require.Greater(t, truncateIdx, -1, "未找到 TRUNCATE 生成")
	require.Less(t, probeIdx, truncateIdx, "源探针必须在 TRUNCATE 之前：否则源侧故障时目标表已被清空")
}
