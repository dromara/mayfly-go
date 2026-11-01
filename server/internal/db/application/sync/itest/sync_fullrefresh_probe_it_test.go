//go:build it

package itest

// 全量刷新的「先探源、再清空」不变量（真库：mysql 源 → pg 目标）。
//
// TRUNCATE 是 DDL（MySQL/PG 下不可回滚）。若先清空目标、再读源，源侧故障会留下
// 「目标表全空 + 任务失败」的不可逆状态。
//
// 触发方式说明（诚实标注）：本用例用「源查询指向不存在的表」制造失败。实测该场景会在
// 更早的列解析阶段就报错，因此它断言的是**端到端不可逆损失不发生**（目标表仍在），
// 而不是"探针这一行代码单独生效"。探针真正覆盖的是「前置校验都通过、但源读取本身失败」
// 的场景（如 TRUNCATE 后权限被回收），单进程 IT 难以稳定模拟，该行由源码顺序守卫
// （TestFullRefreshProbesSourceBeforeTruncate）锁定。

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/domain/entity"
)

func TestITFullRefreshAbortsBeforeTruncateWhenSourceUnreadable(t *testing.T) {
	app, src, dst := setupMode(t)

	// 目标表预置数据：它是否被清空是本次断言的核心
	mustExec(t, dst, fmt.Sprintf("INSERT INTO %s VALUES (100,'old_data',999)", quote(dst, modeItTgtTable)))
	require.Equal(t, 1, countRows(t, dst, modeItTgtTable))

	task := modeTask(entity.DataSyncModeFullRefresh)
	// 源查询指向不存在的表：探针必须在 TRUNCATE 之前执行并失败
	task.DataSQL = fmt.Sprintf("SELECT id, name, status FROM %s_missing", modeItSrcTable)

	err := app.SyncTask(context.Background(), task, src, dst, &entity.DataSyncLog{}, nil)
	require.Error(t, err, "源查询不可执行时必须失败")
	require.Equal(t, 1, countRows(t, dst, modeItTgtTable),
		"目标表必须未被清空：TRUNCATE 不得先于源可读性探针")
}
