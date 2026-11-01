package v112

import (
	"strings"
	"testing"

	sysentity "mayfly-go/internal/sys/domain/entity"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestV1123MigrationIdsAreUniqueAndPrefixed 本批发版约定的机器可验证形式：
// 新增迁移必须全部收口在同目录 v1_12_3.go 的 v1123()，并统一 v1.12.3- 前缀。
//
// 这条守卫专门拦「改错批次」：应用版本升到 v1.12.3 之后，若把新迁移继续追加进已发布的
// v1_12_2.go，gormigrate 仍会执行它，但批次与 ID 前缀就与 Version 脱钩了——
// 之后按版本回溯「这一版改了什么」会得到错误答案。重复 ID 同样致命：
// 记账数对不上时后者永远不执行，表现为「代码改了但库没变」。
func TestV1123MigrationIdsAreUniqueAndPrefixed(t *testing.T) {
	ms := v1123()
	require.NotEmpty(t, ms, "v1.12.3 批次迁移不得为空")

	// 注册断言取本版本入口：同时钉住两件事——批次真的被 Migrations() 接进了链尾，
	// 以及没被漏接或重复接
	registered := map[string]bool{}
	for _, m := range Migrations() {
		registered[m.ID] = true
	}

	ids := map[string]bool{}
	for _, m := range ms {
		require.True(t, strings.HasPrefix(m.ID, "v1.12.3-"),
			"本批新增迁移 ID 必须以 v1.12.3- 开头，实际：%s", m.ID)
		require.False(t, ids[m.ID], "本批迁移 ID 重复：%s", m.ID)
		ids[m.ID] = true
		require.True(t, registered[m.ID], "迁移 %s 必须已注册进迁移链", m.ID)
		require.NotNil(t, m.Migrate, "迁移 %s 必须有执行体", m.ID)
	}
}

// TestV1123ContainsDbInstanceTlsMaterial 钉住本批次承载的 TLS 材料列迁移。
//
// 该列是 DB 实例 TLS 的存储前提：缺了它，保存带证书的实例会因未知列失败，
// 而失败发生在「保存实例」这条低频路径上，很容易在后续重构中被静默删掉。
func TestV1123ContainsDbInstanceTlsMaterial(t *testing.T) {
	const wantID = "v1.12.3-db-instance-tls-material"
	for _, m := range v1123() {
		if m.ID == wantID {
			require.NotNil(t, m.Migrate)
			return
		}
	}
	t.Fatalf("v1.12.3 批次缺少 %s 迁移（DB 实例 TLS 证书材料列）", wantID)
}

// TestV1123AiAgentConfigRowNormalized AiAgentConfig 重复行收敛的行为用例。
//
// 缺陷现场：该配置行由 AI 设置页首次保存创建，permission 为空 → 管理页按权限过滤
// 看不到 → 每次保存都新增一行。重复行会让后端按 key 取首行一直读到最旧值，管理员
// 改的 Agent 配置静默不生效。本用例锁住收敛结果：一条存活行、权限归一到 admin,、
// 取值继承最新一次保存、重复行软删，且重复执行幂等。
func TestV1123AiAgentConfigRowNormalized(t *testing.T) {
	db := openCronMigDB(t)
	require.NoError(t, db.AutoMigrate(&sysentity.Config{}))

	const key = "AiAgentConfig"
	insertRow := func(id uint64, value, permission string) {
		require.NoError(t, db.Exec(
			"INSERT INTO t_sys_config (id, name, `key`, params, value, permission, create_time, update_time, creator_id, creator, modifier_id, modifier, is_deleted) VALUES (?, '', ?, '', ?, ?, '2026-10-21 22:00:00', '2026-10-21 22:00:00', 1, 'admin', 1, 'admin', 0)",
			id, key, value, permission,
		).Error)
	}
	insertRow(11, `{}`, "")
	insertRow(12, `{"toolSearchThreshold":30}`, "")
	insertRow(13, `{"toolSearchThreshold":50}`, "")

	migrate := findMigration(t, "v1.12.3-ai-agent-config-permission-dedupe")
	require.NoError(t, migrate(db))
	// 幂等：重复执行不得再建行、不得改变结果
	require.NoError(t, migrate(db))

	var rows []sysentity.Config
	require.NoError(t, db.Where("`key` = ? AND is_deleted = 0", key).Order("id ASC").Find(&rows).Error)
	require.Len(t, rows, 1, "重复配置行必须收敛为一条存活行")
	require.Equal(t, uint64(11), rows[0].Id, "保留最小 id：后端按 key 取首行的稳定目标")
	require.Equal(t, "admin,", rows[0].Permission, "权限必须归一到管理类配置口径，否则管理页仍然看不到")
	require.JSONEq(t, `{"toolSearchThreshold":50}`, rows[0].Value, "必须继承最新一次保存的取值")
	require.Equal(t, "ai.settings.agentConfig", rows[0].Name, "空显示名要用 i18n key 补齐")

	var deleted int64
	require.NoError(t, db.Model(&sysentity.Config{}).Where("is_deleted = 1").Count(&deleted).Error)
	require.Equal(t, int64(2), deleted, "重复行应软删而不是物理删除")
}

// TestV1123AiAgentConfigRowCreatedWhenMissing 行不存在时按同一口径补建（全新库可用）
func TestV1123AiAgentConfigRowCreatedWhenMissing(t *testing.T) {
	db := openCronMigDB(t)
	require.NoError(t, db.AutoMigrate(&sysentity.Config{}))

	migrate := findMigration(t, "v1.12.3-ai-agent-config-permission-dedupe")
	require.NoError(t, migrate(db))
	require.NoError(t, migrate(db))

	var rows []sysentity.Config
	require.NoError(t, db.Where("`key` = ? AND is_deleted = 0", "AiAgentConfig").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "admin,", rows[0].Permission)
	require.JSONEq(t, `{}`, rows[0].Value)
	require.NotZero(t, rows[0].GetId(), "建行必须落到基座字段，不能因 NOT NULL 列缺值失败")
}

// findMigration 按 ID 取迁移执行体（与 openCronMigDB 同一就地取用范式）
func findMigration(t *testing.T, id string) func(*gorm.DB) error {
	t.Helper()
	for _, m := range Migrations() {
		if m.ID == id {
			require.NotNil(t, m.Migrate)
			return m.Migrate
		}
	}
	t.Fatalf("migration %s not found", id)
	return nil
}
