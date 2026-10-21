package v112

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"mayfly-go/pkg/utils/collx"

	agentopsentity "mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/ai/pkg/headless"
	alertentity "mayfly-go/internal/alert/domain/entity"
	"mayfly-go/migration/migrations/initdb"
	"mayfly-go/migration/migrations/migtest"
	v110 "mayfly-go/migration/migrations/v1_10"
	v111 "mayfly-go/migration/migrations/v1_11"
	v19 "mayfly-go/migration/migrations/v1_9"
	"mayfly-go/pkg/model"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestV1122MigrationIdsAreUniqueAndPrefixed 本批发版约定的机器可验证形式：
// 新增迁移必须全部收口在同目录 v1_12_2.go 的 v1122()，并统一 v1.12.2- 前缀。
// 重编号（v1.12.1、v1.12.3~7 一律归并为 v1.12.2）过程中最容易犯的是复制粘贴出重复 ID——
// 重复 ID 会让记账数对不上、后者永远不执行，是「代码改了但库没变」的典型成因
func TestV1122MigrationIdsAreUniqueAndPrefixed(t *testing.T) {
	ms := v1122()
	require.NotEmpty(t, ms, "v1.12.2 批次迁移不得为空")

	// 注册断言取本版本入口：同时钉住两件事——批次真的被 Migrations() 接进了链尾，
	// 以及没被漏接或重复接（漏接时这里查不到 ID，重复接时上一条幂等断言拦不住）
	registered := map[string]bool{}
	for _, m := range Migrations() {
		registered[m.ID] = true
	}

	ids := map[string]bool{}
	for _, m := range ms {
		require.True(t, strings.HasPrefix(m.ID, "v1.12.2-"),
			"本批新增迁移 ID 必须以 v1.12.2- 开头，实际：%s", m.ID)
		require.False(t, ids[m.ID], "本批迁移 ID 重复：%s", m.ID)
		ids[m.ID] = true
		require.True(t, registered[m.ID], "迁移 %s 必须已注册进迁移链", m.ID)
		require.NotNil(t, m.Migrate, "迁移 %s 必须有执行体", m.ID)
	}
}

func columnsOf(t *testing.T, db *gorm.DB, dest any) map[string]bool {
	t.Helper()
	types, err := db.Migrator().ColumnTypes(dest)
	require.NoError(t, err)
	out := make(map[string]bool, len(types))
	for _, ct := range types {
		out[strings.ToLower(ct.Name())] = true
	}
	return out
}

// TestDisplayColumnsCollapseBackfills 验证 v1.12.2 的升级路径：旧列值必须先回填进 extra 再删列。
//
// 这是本步唯一会被静默破坏的地方：若先删列或忘记回填，历史任务/运行的负责人与备注会在界面上
// 凭空变空且不报错（无报错的丢数据）。也断言旧列确实被删，避免它们以 NULL 残留在库里。
// Run 与 AgentTask 各覆盖一份：两个实体的钩子对各自实现（键集不同，漏一边不会被对方用例拦下）
func TestDisplayColumnsCollapseBackfills(t *testing.T) {
	db := migtest.OpenSqlite(t)
	// 步骤体会遍历 Run/Task/Trigger/Proposal 四张表，夹具须全建出来：真实链路上它们在本步之前
	// 已建表，缺表时 Migrator 直接报错而不是跳过
	require.NoError(t, db.AutoMigrate(
		&agentopsentity.Run{}, &agentopsentity.AgentTask{},
		&agentopsentity.Trigger{}, &agentopsentity.Proposal{}))

	// 模拟「升级前的库」：把已废弃的展示列加回来，并把 extra 置为 NULL（真实老库就是没有该列）
	require.NoError(t, db.Exec("ALTER TABLE t_aiops_task ADD COLUMN owner_username TEXT").Error)
	require.NoError(t, db.Exec("ALTER TABLE t_aiops_task ADD COLUMN remark TEXT").Error)
	require.NoError(t, db.Exec("ALTER TABLE t_aiops_run ADD COLUMN owner_username TEXT").Error)
	require.NoError(t, db.Exec("ALTER TABLE t_aiops_run ADD COLUMN feedback_note TEXT").Error)

	// 有值行：旧列存 owner/remark，extra 保持 NULL —— 回填必须覆盖这一形态
	// 审计列（create_time 等）是 not null 且由 FillBaseInfo 填，测试无登录上下文，按后台任务口径回落系统账号
	withValues := &agentopsentity.AgentTask{Name: "慢查询巡检", Key: "display-extra-owner"}
	withValues.FillBaseInfo(model.SysAccount)
	require.NoError(t, db.Create(withValues).Error)
	// 空值行：remark 为空串，回填时不该凭空造键
	blankRemark := &agentopsentity.AgentTask{Name: "空备注", Key: "display-extra-blank"}
	blankRemark.FillBaseInfo(model.SysAccount)
	require.NoError(t, db.Create(blankRemark).Error)

	legacyRun := &agentopsentity.Run{
		RunNo: "rn-display-extra", TriggerType: agentopsentity.TriggerTypeManual,
		ProfileId: headless.ProfileDiagnostician, Status: agentopsentity.RunStatusSucceeded,
		OwnerAccountId: 1, IdempotencyKey: "display:extra",
	}
	legacyRun.FillBaseInfo(model.SysAccount)
	require.NoError(t, db.Create(legacyRun).Error)

	legacyValues := func(table string, id uint64, owner, legacyValue string) {
		require.NoError(t, db.Table(table).Where("id = ?", id).
			UpdateColumns(map[string]any{"owner_username": owner, legacyColumn(table): legacyValue, "extra": nil}).Error)
	}
	legacyValues("t_aiops_task", withValues.Id, "admin", "内置模板")
	legacyValues("t_aiops_task", blankRemark.Id, "ops", "")
	legacyValues("t_aiops_run", legacyRun.Id, "admin", "结论可直接复用")

	cols := columnsOf(t, db, &agentopsentity.AgentTask{})
	require.True(t, cols["owner_username"], "夹具应已还原旧列形态")

	// 跑真实步骤体：回填 → 删列（extra 列由迁移步骤前的 AutoMigrate 保证存在）
	require.NoError(t, collapseAgentOpsDisplayColumns(db))

	for _, legacy := range []struct {
		dest    any
		table   string
		columns []string
	}{
		{&agentopsentity.AgentTask{}, "t_aiops_task", []string{"owner_username", "remark"}},
		{&agentopsentity.Run{}, "t_aiops_run", []string{"owner_username", "feedback_note"}},
	} {
		after := columnsOf(t, db, legacy.dest)
		require.True(t, after["extra"], "extra 列必须存在（展示字段唯一落库处）")
		for _, column := range legacy.columns {
			require.False(t, after[column], "表 %s 的 %s 应已被删除，否则旧列以 NULL 残留", legacy.table, column)
		}
	}

	// 用实体读回：AfterFind 必须从 extra 还原出瞬态字段
	var got agentopsentity.AgentTask
	require.NoError(t, db.First(&got, withValues.Id).Error)
	require.Equal(t, "admin", got.OwnerUsername, "旧列值必须回填进 extra 并可读回")
	require.Equal(t, "内置模板", got.Remark)

	var blank agentopsentity.AgentTask
	require.NoError(t, db.First(&blank, blankRemark.Id).Error)
	require.Equal(t, "ops", blank.OwnerUsername)
	require.Empty(t, blank.Remark, "空值不应被写成 extra 键")

	var gotRun agentopsentity.Run
	require.NoError(t, db.First(&gotRun, legacyRun.Id).Error)
	require.Equal(t, "admin", gotRun.OwnerUsername)
	require.Equal(t, "结论可直接复用", gotRun.FeedbackNote, "feedback_note 旧列值必须走 ExtraKeyFeedbackNote 回填")

	// 幂等：再跑一次不报错、值不丢（迁移按 ID 只记账一次，但重跑安全是硬要求）
	require.NoError(t, collapseAgentOpsDisplayColumns(db))
	var again agentopsentity.AgentTask
	require.NoError(t, db.First(&again, withValues.Id).Error)
	require.Equal(t, "admin", again.OwnerUsername)
	var againRun agentopsentity.Run
	require.NoError(t, db.First(&againRun, legacyRun.Id).Error)
	require.Equal(t, "结论可直接复用", againRun.FeedbackNote)
}

// legacyColumn 各表待回填的展示列名（与 agentOpsDisplayColumns 中的非 owner 列同源）
func legacyColumn(table string) string {
	if table == "t_aiops_run" {
		return "feedback_note"
	}
	return "remark"
}

// TestAgentOpsExtraKeysMatchDomain 回填键必须正是实体读回用的键（迁移与 domain 常量同源，
// 拼错键的表现是「值在 JSON 里但界面读不到」，没有报错，只能靠这条断言拦下）
func TestAgentOpsExtraKeysMatchDomain(t *testing.T) {
	for _, spec := range agentOpsDisplayColumns() {
		switch spec.extraKey {
		case agentopsentity.ExtraKeyOwnerUsername, agentopsentity.ExtraKeyRemark, agentopsentity.ExtraKeyFeedbackNote:
		default:
			t.Fatalf("表 %s 的回填键 %q 不在 domain 常量集合内，读回会取不到", spec.table, spec.extraKey)
		}
	}
}

// TestTargetResourcesFoldsIntoInstruction 验证 v1.12.2 的升级路径：旧「目标资源」列的值必须先
// 折进任务指令，再删列。
//
// 只删列不折值的话，已有任务下次运行时就没有目标了：模型会转而自己盘点甚至猜资源，属于不报错的
// 行为改变（表单也不再提交该字段，不会覆盖回来），所以这一步必须被用例钉住
func TestTargetResourcesFoldsIntoInstruction(t *testing.T) {
	db := migtest.OpenSqlite(t)
	require.NoError(t, db.AutoMigrate(&agentopsentity.AgentTask{}))

	// 还原「升级前的库」：目标资源列存在且已有值
	require.NoError(t, db.Exec("ALTER TABLE t_aiops_task ADD COLUMN target_resources TEXT").Error)

	withTarget := &agentopsentity.AgentTask{Name: "DB 慢查询巡检", Key: "targets-with", Instruction: "检查慢查询并产出报告"}
	withTarget.FillBaseInfo(model.SysAccount)
	require.NoError(t, db.Create(withTarget).Error)
	require.NoError(t, db.Table("t_aiops_task").Where("id = ?", withTarget.Id).
		UpdateColumn("target_resources", `{"dbId":1}`).Error)

	blank := &agentopsentity.AgentTask{Name: "无目标任务", Key: "targets-blank", Instruction: "汇总今日告警"}
	blank.FillBaseInfo(model.SysAccount)
	require.NoError(t, db.Create(blank).Error)
	require.NoError(t, db.Table("t_aiops_task").Where("id = ?", blank.Id).
		UpdateColumn("target_resources", "").Error)

	require.NoError(t, foldAgentOpsTargetResourcesIntoInstruction(db))

	// 存量值折进指令（措辞沿用原运行输入注入文本，模型侧语义不变）
	var got agentopsentity.AgentTask
	require.NoError(t, db.First(&got, withTarget.Id).Error)
	require.Contains(t, got.Instruction, "检查慢查询并产出报告", "原指令不得被覆盖")
	require.Contains(t, got.Instruction, `{"dbId":1}`, "目标资源必须折进指令")

	// 空值不折：不该给纯文本指令凭空追加一行
	var gotBlank agentopsentity.AgentTask
	require.NoError(t, db.First(&gotBlank, blank.Id).Error)
	require.Equal(t, "汇总今日告警", gotBlank.Instruction)

	// 列必须真的消失（Migrator().DropColumn 在本驱动上是静默空操作）
	require.False(t, columnsOf(t, db, &agentopsentity.AgentTask{})["target_resources"], "target_resources 应已被删除")

	// 幂等：列已不存在时再跑一次安全无操作
	require.NoError(t, foldAgentOpsTargetResourcesIntoInstruction(db))
	var again agentopsentity.AgentTask
	require.NoError(t, db.First(&again, withTarget.Id).Error)
	require.Contains(t, again.Instruction, `{"dbId":1}`, "重跑不得丢值")
	require.NotContains(t, again.Instruction, `{"dbId":1}`+"\n"+`{"dbId":1}`, "重跑不得重复追加")
}

// TestBackfillAgentOpsRunRoleFromTitle 存量运行的形态必须从标题前缀搬进 Extra 并把前缀剥掉。
//
// 这条链有三处容易写坏，且写坏后都不报错：① 漏剥前缀 → 前端拿到 runRole 又看到 `[编排] `，标签
// 重复；② 覆盖整列 Extra → 抹掉该行已有的一次性工具许可/工具级失败记录（比少个标签严重得多）；
// ③ 误伤用户自建标题（恰好以 `[` 开头但不是本模块形态）或脏 Extra 行被半改。
//
// 断言取**落库 JSON 原文**而不是经实体读回：Extra 无 Scan 实现，反序列化行为随方言驱动而异
// （sqlite 下读不回），迁移的职责只是把正确的 JSON 写进列，钩子层的往返由 entity 用例负责
func TestBackfillAgentOpsRunRoleFromTitle(t *testing.T) {
	db := migtest.OpenSqlite(t)
	require.NoError(t, db.AutoMigrate(&agentopsentity.Run{}))

	insert := func(id uint64, title string, extra collx.M) {
		run := &agentopsentity.Run{
			Title: title, TriggerType: agentopsentity.TriggerTypeManual,
			ProfileId: headless.ProfileDiagnostician, Status: agentopsentity.RunStatusSucceeded,
			RunNo: fmt.Sprintf("rn-%d", id), IdempotencyKey: fmt.Sprintf("mig-role-backfill-%d", id),
		}
		run.Id = id
		// 审计时间戳的自动填充依赖上下文登录态，迁移测试里没有，必须显式给（create_time 是 not null）
		now := time.Now()
		run.CreateTime, run.UpdateTime = &now, &now
		if extra != nil {
			run.ExtraData = model.ExtraData{Extra: extra}
		}
		require.NoError(t, db.Create(run).Error)
	}

	insert(1, "[编排步骤 2] 清理临时表", collx.M{"toolErrors": []string{"boom"}}) // 前缀含可变序号 + 已有 Extra
	insert(2, "[编排] 定位慢查询", nil)
	insert(3, "[评测回放] 巡检", collx.M{})
	insert(4, "[校验] 重启 nginx", nil)
	insert(5, "每日运维日报", nil)        // 普通运行：不得被动
	insert(6, "[自取标题] 别动我", nil)    // 用户自建方括号标题：不属本模块形态
	insert(7, "[执行] 重启 nginx", nil) // 脏 Extra：整行跳过
	require.NoError(t, db.Table("t_aiops_run").Where("id = ?", 7).Update("extra", "not-a-json").Error)

	require.NoError(t, backfillAgentOpsRunRoleFromTitle(db))

	type got struct {
		Id    uint64 `gorm:"column:id"`
		Title string `gorm:"column:title"`
		Extra string `gorm:"column:extra"`
	}
	load := func(id uint64) got {
		var g got
		require.NoError(t, db.Table("t_aiops_run").Select("id, title, extra").Where("id = ?", id).Take(&g).Error)
		return g
	}

	// ① 前缀剥掉、形态进 Extra、原有 Extra 键一个不少
	r1 := load(1)
	require.Equal(t, "清理临时表", r1.Title)
	require.Contains(t, r1.Extra, `"runRole":"orch_step"`)
	require.Contains(t, r1.Extra, "toolErrors", "回填不得覆盖既有 Extra 键")

	require.Equal(t, "定位慢查询", load(2).Title)
	require.Contains(t, load(2).Extra, `"runRole":"`+agentopsentity.RunRoleOrchParent+`"`)
	require.Contains(t, load(3).Extra, `"runRole":"`+agentopsentity.RunRoleEvalReplay+`"`)
	require.Contains(t, load(4).Extra, `"runRole":"`+agentopsentity.RunRoleVerify+`"`)

	// ② 未命中形态前缀的行：标题与 Extra 都不动
	require.Equal(t, "每日运维日报", load(5).Title)
	require.NotContains(t, load(5).Extra, agentopsentity.ExtraKeyRunRole)
	require.Equal(t, "[自取标题] 别动我", load(6).Title)
	require.NotContains(t, load(6).Extra, agentopsentity.ExtraKeyRunRole)

	// ③ 脏 Extra 整行跳过：不能只改标题留下「标题没了但形态也没了」的半截结果
	r7 := load(7)
	require.Equal(t, "[执行] 重启 nginx", r7.Title, "脏 Extra 行必须整行跳过")
	require.Equal(t, "not-a-json", r7.Extra)
}

// TestAlertNotifyLogPolicyTypeColumnIsAdded 升级库必须补出 policy_type 列。
//
// 不补的后果是硬的：写入侧是裸 Create（全列 INSERT），而建表步 v1.12.0-alert-tables
// 在已跑过它的库上不会重跑，于是升级库上每次记通知日志都报 Unknown column 'policy_type'，
// 而那个错误只被降为一条 Warn——通知日志整批静默丢失。
//
// 同时钉住存量行的语义：本列上线前的历史行无法回溯来源，policy_type 必须是 0（无归因），
// 不能被猜成「通知策略」——猜错比留空更坑，它会把排查引到一张根本不相关的表上
func TestAlertNotifyLogPolicyTypeColumnIsAdded(t *testing.T) {
	db := migtest.OpenSqlite(t)
	require.NoError(t, db.AutoMigrate(&alertentity.AlertNotifyLog{}))

	// 造出升级前的库形态：先删列再插存量行（删列语法 sqlite 3.35+ 与 mysql 均原生支持）。
	// 存量行的 policy_id=11 取自真机：它在 t_alert_notify_policy 查无此行，实为升级策略 11
	require.NoError(t, db.Exec("ALTER TABLE t_alert_notify_log DROP COLUMN policy_type").Error)
	require.False(t, db.Migrator().HasColumn(&alertentity.AlertNotifyLog{}, "policy_type"),
		"用例前提：老库不该有该列，否则测的不是升级路径")
	require.NoError(t, db.Exec(
		"INSERT INTO t_alert_notify_log (event_id, policy_id, channel_id, channel_name, status, error_msg, send_time) VALUES (?,?,?,?,?,?,?)",
		100, 11, 1, "test-企微", 1, "", time.Now()).Error)

	var step *gormigrate.Migration
	for _, m := range v1122() {
		if m.ID == "v1.12.2-alert-notify-log-policy-type-column" {
			step = m
		}
	}
	require.NotNil(t, step, "未找到 policy_type 迁移步")

	// 跑两次：迁移失败会重试未记账项，重跑安全是硬要求（AutoMigrate 也不得报「列已存在」）
	for i := 0; i < 2; i++ {
		require.NoError(t, step.Migrate(db), "第 %d 次执行必须成功", i+1)
	}
	require.True(t, db.Migrator().HasColumn(&alertentity.AlertNotifyLog{}, "policy_type"))

	// 存量行：policy_id 原值不动，来源置未知
	var legacy alertentity.AlertNotifyLog
	require.NoError(t, db.Where("event_id = ?", 100).First(&legacy).Error)
	require.Equal(t, uint64(11), legacy.PolicyId)
	require.Equal(t, alertentity.NotifyPolicyTypeUnknown, legacy.PolicyType,
		"历史行来源无法回溯，只能留 0；猜成通知策略会把排查引到错误的表上")

	// 升级后写入侧的裸 Create（全列 INSERT）必须真的能落库——这才是本步要防的失败
	require.NoError(t, db.Create(&alertentity.AlertNotifyLog{
		EventId: 101, PolicyId: 11, PolicyType: alertentity.NotifyPolicyTypeEscalation,
		ChannelId: 1, ChannelName: "test-企微", Status: 1, SendTime: time.Now(),
	}).Error)
	var fresh alertentity.AlertNotifyLog
	require.NoError(t, db.Where("event_id = ?", 101).First(&fresh).Error)
	require.Equal(t, alertentity.NotifyPolicyTypeEscalation, fresh.PolicyType,
		"升级策略与通知策略主键空间独立且已重叠，类型必须随 id 一起落库 才可溯源")
}

// 升级前已存在、且拥有对应菜单的自定义角色，必须一并拿到新补的基础权限码。
// 只授 COMMON 角色（role_id=1）会让其他角色的首页入口在升级后凭空消失
func TestV1122BasePermissionCodesGrantedToExistingMenuRoles(t *testing.T) {
	db := migtest.OpenSqlite(t)

	// 先跑到 v1.11，模拟升级前的存量库
	require.NoError(t, migtest.NewChainMigrator(db, migtest.Concat(
		initdb.Migrations, v19.Migrations, v110.Migrations, v111.Migrations,
	)).Migrate())

	// 自定义角色：拥有「我的流程」菜单（1709103180），但自然还没有新基础码
	const customRoleId = 9001
	require.NoError(t, db.Exec(
		"INSERT INTO t_sys_role_resource (role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (?, 1709103180, 1, 'admin', datetime('now'), 0, NULL)",
		customRoleId,
	).Error)

	// 再跑 v1.12 全链（含本批新增的基础权限码）
	require.NoError(t, migtest.NewChainMigrator(db, Migrations()).Migrate())

	var codes []string
	require.NoError(t, db.Table("t_sys_resource").
		Where("id IN (1777000001, 1777000002, 1777000003) AND is_deleted = 0").
		Pluck("code", &codes).Error)
	require.ElementsMatch(t, []string{"resource:base", "flow:base", "flow:task:base"}, codes)

	var count int64
	// 拥有父菜单的角色拿到新码
	require.NoError(t, db.Table("t_sys_role_resource").
		Where("role_id = ? AND resource_id = 1777000002 AND is_deleted = 0", customRoleId).
		Count(&count).Error)
	require.EqualValues(t, 1, count)

	// 没有父菜单的角色不应被授予
	require.NoError(t, db.Table("t_sys_role_resource").
		Where("role_id = ? AND resource_id = 1777000001 AND is_deleted = 0", customRoleId).
		Count(&count).Error)
	require.EqualValues(t, 0, count)
}
