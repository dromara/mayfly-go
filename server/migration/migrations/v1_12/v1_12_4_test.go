package v112

import (
	"strings"
	"testing"

	dbentity "mayfly-go/internal/db/domain/entity"

	"github.com/stretchr/testify/require"
)

// TestV1124MigrationIdsAreUniqueAndPrefixed 本批发版约定的机器可验证形式：
// 新增迁移必须全部收口在同目录 v1_12_4.go 的 v1124()，并统一 v1.12.4- 前缀。
//
// 与 v1.12.3 同一判据：应用版本已升到 v1.12.4，若把新迁移继续追加进已发布的
// v1_12_3.go，gormigrate 照样执行，但批次与 ID 前缀就和 Version 脱钩了。
func TestV1124MigrationIdsAreUniqueAndPrefixed(t *testing.T) {
	ms := v1124()
	require.NotEmpty(t, ms, "v1.12.4 批次迁移不得为空")

	registered := map[string]bool{}
	for _, m := range Migrations() {
		registered[m.ID] = true
	}

	ids := map[string]bool{}
	for _, m := range ms {
		require.True(t, strings.HasPrefix(m.ID, "v1.12.4-"),
			"本批新增迁移 ID 必须以 v1.12.4- 开头，实际：%s", m.ID)
		require.False(t, ids[m.ID], "本批迁移 ID 重复：%s", m.ID)
		ids[m.ID] = true
		require.True(t, registered[m.ID], "迁移 %s 必须已注册进迁移链", m.ID)
		require.NotNil(t, m.Migrate, "迁移 %s 必须有执行体", m.ID)
	}
}

// TestV1124CreatesDbAiTermIdempotent 业务词典表必须建得出来、重复执行不报错、列名映射正确。
//
// 重复执行这条不是形式：gormigrate 只在首次升级跑一次，但「全新库 + 已发布批次重放」
// 与 IT 环境的 scratch 库都会二次进入这里，AutoMigrate 不幂等就会在升级链上直接失败。
func TestV1124CreatesDbAiTermIdempotent(t *testing.T) {
	db := openCronMigDB(t)
	migrate := findMigration(t, "v1.12.4-db-ai-term")
	require.NoError(t, migrate(db))
	require.NoError(t, migrate(db))

	require.True(t, db.Migrator().HasTable(&dbentity.DbAiTerm{}), "词典表必须存在")
	// schema/tables 在部分方言里是关键字，列名必须走 schema_name / table_names
	for _, column := range []string{"db_id", "schema_name", "term", "table_names", "details"} {
		require.True(t, db.Migrator().HasColumn(&dbentity.DbAiTerm{}, column), "缺少列 %s", column)
	}

}
