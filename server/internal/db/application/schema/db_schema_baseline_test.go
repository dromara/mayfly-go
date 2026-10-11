package schema

// 漂移判定与明细截断的纯逻辑单测。
//
// 这两处是「用户能不能自证漂移」的关键：指纹一致却报漂移、或明细被整句丢掉，
// 都会让通知在几次之后被人关掉。这里不碰仓库/采集，只钉判定与截断本身。

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/i18n"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateDrift(t *testing.T) {
	ctx := context.Background()
	app := &SchemaBaselineAppImpl{}

	t.Run("指纹一致即无漂移，且清掉旧明细", func(t *testing.T) {
		baseline := &entity.DbSchemaBaseline{Fingerprint: "same", ColumnCount: 3, DriftDetail: "旧文案"}
		drifted, detail := app.evaluateDrift(ctx, baseline, TableSummary{Fingerprint: "same", ColumnCount: 3})
		assert.False(t, drifted)
		assert.Empty(t, detail, "指纹一致时必须清空明细，否则结构已回到基线却仍显示旧漂移文案")
	})

	t.Run("基线无指纹即新增表", func(t *testing.T) {
		baseline := &entity.DbSchemaBaseline{Fingerprint: ""}
		drifted, detail := app.evaluateDrift(ctx, baseline, TableSummary{Fingerprint: "x", ColumnCount: 2})
		assert.True(t, drifted)
		assert.Equal(t, i18n.TC(ctx, imsg.SchemaDriftTableAdded), detail)
	})

	t.Run("列数变化给出可读明细而非仅「结构变了」", func(t *testing.T) {
		baseline := &entity.DbSchemaBaseline{
			Fingerprint: "old", ColumnCount: 8, PrimaryKeys: "id", IndexCount: 1,
		}
		current := TableSummary{
			Fingerprint: "new", ColumnCount: 9, PrimaryKeys: "id",
			PrimaryKeysAvailable: true, IndexCount: 1, IndexesAvailable: true,
		}
		drifted, detail := app.evaluateDrift(ctx, baseline, current)
		require.True(t, drifted)
		assert.Contains(t, detail, "8")
		assert.Contains(t, detail, "9", "明细要能说清「列数 8→9」，而不是一句笼统的结构变化")
	})

	t.Run("指纹变但列数/主键/索引都没变时归因为列定义变化", func(t *testing.T) {
		baseline := &entity.DbSchemaBaseline{
			Fingerprint: "old", ColumnCount: 5, PrimaryKeys: "id", IndexCount: 2,
		}
		current := TableSummary{
			Fingerprint: "new", ColumnCount: 5, PrimaryKeys: "id",
			PrimaryKeysAvailable: true, IndexCount: 2, IndexesAvailable: true,
		}
		drifted, detail := app.evaluateDrift(ctx, baseline, current)
		require.True(t, drifted)
		assert.Equal(t, i18n.TC(ctx, imsg.SchemaDriftColumnDefs), detail,
			"拼不出更细明细时也不能返回空，否则已确认的漂移被说成没变")
	})
}

func TestTruncateDetail(t *testing.T) {
	t.Run("未超限原样返回", func(t *testing.T) {
		short := "列数 8→9"
		assert.Equal(t, short, truncateDetail(short))
	})

	t.Run("超限按 rune 截断并标注省略号", func(t *testing.T) {
		long := strings.Repeat("长", BaselineDriftDetailMax+50)
		got := truncateDetail(long)
		assert.Equal(t, BaselineDriftDetailMax, len([]rune(got)), "截断后 rune 数应正好等于上限")
		assert.True(t, strings.HasSuffix(got, "…"), "截断必须以省略号标注，避免被当成完整明细")
	})

	t.Run("恰好等于上限不截断", func(t *testing.T) {
		exact := strings.Repeat("a", BaselineDriftDetailMax)
		assert.Equal(t, exact, truncateDetail(exact))
	})
}
