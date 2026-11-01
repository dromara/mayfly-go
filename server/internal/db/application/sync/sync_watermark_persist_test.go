package sync

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWatermarkPersistsOnlyAfterFullSuccess 增量水位只允许在整轮成功后推进一次。
//
// 历史缺陷：批间就把水位推进到「本批末行的 upd 值」。配合 `> V` 续跑，
// 一旦在批后失败/停止/被抢占，与 V 同值但位于后续批次尚未写入的行会被永久跳过
// （UpdField 非唯一时必然发生，如秒级时间戳），表现为"任务成功但少同步了数据"。
//
// 现在的语义是「至少一次」：中断后整轮重放，重复由冲突策略/主键暴露。
// 本用例按源码结构守卫：persistUpdFieldVal 只出现一次，且在「整轮成功」注释之后。
func TestWatermarkPersistsOnlyAfterFullSuccess(t *testing.T) {
	content, err := os.ReadFile("db_data_sync_exec.go")
	require.NoError(t, err)
	source := string(content)

	require.Equal(t, 1, strings.Count(source, "persistUpdFieldVal(ctx, task)"),
		"水位推进只允许有一处（整轮成功后）；批间推进会让中断后的续跑跳过同值行")

	persist := strings.Index(source, "persistUpdFieldVal(ctx, task)")
	fullSuccess := strings.Index(source, "整轮成功：无条件推进一次水位")
	require.Greater(t, fullSuccess, -1, "必须保留「整轮成功才推进」的语义说明")
	require.Greater(t, persist, fullSuccess, "水位推进必须位于整轮成功之后")

	// 批间批次块内不得再出现水位推进（按注释锚点划分）
	require.NotContains(t, source, "watermark persist failed (batch #",
		"批间不应再有水位推进的分支")
}
