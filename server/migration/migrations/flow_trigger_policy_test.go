package migrations

import (
	"strings"
	"testing"

	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// triggerPolicyMigration 取出触发策略迁移，供测试直接执行
func triggerPolicyMigration(t *testing.T) func(*gorm.DB) error {
	t.Helper()
	for _, m := range V1_12() {
		if m.ID == "v1.12.0-flow-trigger-policy" {
			return m.Migrate
		}
	}
	t.Fatal("flow trigger policy migration not found")
	return nil
}

func openFlowMigDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	return db
}

// legacyProcdef 改造前的流程定义表结构：触发条件是一段 Go 模板文本，且没有 trigger_policy 列
type legacyProcdef struct {
	model.Model

	Name      string  `gorm:"size:150"`
	DefKey    string  `gorm:"not null;size:100"`
	FlowDef   string  `gorm:"type:text"`
	Status    int8    `gorm:"not null"`
	Condition *string `gorm:"type:text"`
	Remark    *string `gorm:"size:255"`
}

func (legacyProcdef) TableName() string { return "t_flow_procdef" }

// createLegacyProcdefTable 建出旧表并插入一条历史数据
func createLegacyProcdefTable(t *testing.T, db *gorm.DB, condition string) {
	t.Helper()
	if err := db.AutoMigrate(&legacyProcdef{}); err != nil {
		t.Fatalf("create legacy procdef table failed: %v", err)
	}
	// 审计列均为 NOT NULL，直插历史数据时必须手动补齐
	now := time.Now()
	legacy := &legacyProcdef{Name: "old flow", DefKey: "old_key", Status: 1, Condition: &condition}
	legacy.Id = 1
	legacy.CreateTime = &now
	legacy.UpdateTime = &now
	legacy.Creator = "admin"
	legacy.Modifier = "admin"
	if err := db.Create(legacy).Error; err != nil {
		t.Fatalf("insert legacy row failed: %v", err)
	}
}

// defaultRulePackKeys 默认规则包应包含的检查项。
// 逐条校验（字段/参数/操作符合法性）在 db 与 redis 各自的包内测试覆盖，那里才有完整注册表
var defaultRulePackKeys = []string{
	"db.dml-requires-approval", "db.dml-without-where",
	"redis.write-cmd", "redis.dangerous-cmd",
}

func assertDefaultRulePack(t *testing.T, policy *flowentity.TriggerPolicy) {
	t.Helper()
	configured := map[string]bool{}
	for _, check := range policy.Checks {
		if check.BizType == "" {
			t.Fatalf("the migrated check %q must declare its biz type", check.Key)
		}
		if check.Severity != flowentity.SeverityRequired {
			t.Fatalf("the migrated check %q must require approval, got %v", check.Key, check.Severity)
		}
		configured[check.Key] = true
	}
	for _, want := range defaultRulePackKeys {
		if !configured[want] {
			t.Fatalf("the default rule pack must contain %q, got %v", want, configured)
		}
	}
}

func legacyConditionText() string {
	return `{{ if eq .bizType "db_sql_exec_flow" }}{{ if and (ne .param.stmtType "select") (ne .param.stmtType "read") }}1{{ end }}{{ end }}`
}

// 迁移后：旧列被删除、存量行拿到可校验的默认规则包、且不会覆盖已配置的策略
func TestFlowTriggerPolicyMigration(t *testing.T) {
	db := openFlowMigDB(t)
	createLegacyProcdefTable(t, db, legacyConditionText())

	if err := triggerPolicyMigration(t)(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	migrator := db.Migrator()
	if migrator.HasColumn(&flowentity.Procdef{}, "condition") {
		t.Fatalf("the legacy condition column must be dropped")
	}
	if !migrator.HasColumn(&flowentity.Procdef{}, "trigger_policy") {
		t.Fatalf("the trigger_policy column must exist after the migration")
	}

	var stored flowentity.Procdef
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("reload the migrated row failed: %v", err)
	}
	policy := stored.TriggerPolicy
	if policy == nil {
		t.Fatalf("the legacy row must receive the default rule pack")
	}
	assertDefaultRulePack(t, policy)
	// 迁移不得替存量数据加严：兜底保持「不处置」，治理强度只来自下面那四条等价搬来的显式规则。
	// 「需审批」兜底在没有审批通道的场景会被求值期收敛成禁止执行，等于升级当晚把机器命令全拦死
	if policy.DefaultSeverity != flowentity.SeverityDisabled {
		t.Fatalf("the migration must not tighten legacy governance via the fallback severity, got %v", policy.DefaultSeverity)
	}
	if len(policy.Checks) == 0 {
		t.Fatalf("the default rule pack must contain checks")
	}

	// 旧模板文本不得残留在任何字段里（否则等于保留了第二真源）
	var raw string
	if err := db.Table("t_flow_procdef").Select("trigger_policy").Where("id = 1").Scan(&raw).Error; err != nil {
		t.Fatalf("read the raw policy failed: %v", err)
	}
	if strings.Contains(raw, "{{") || strings.Contains(raw, ".param.") {
		t.Fatalf("the legacy go template must not be carried into the new model, got %s", raw)
	}
}

// 重复执行必须安全：已配置策略的行不能被默认规则包覆盖
func TestFlowTriggerPolicyMigrationIsIdempotent(t *testing.T) {
	db := openFlowMigDB(t)
	createLegacyProcdefTable(t, db, legacyConditionText())
	migrate := triggerPolicyMigration(t)

	if err := migrate(db); err != nil {
		t.Fatalf("first migration run failed: %v", err)
	}
	custom := flowentity.NewTriggerPolicy(flowentity.SeverityDisabled)
	// 单列 Update 不走实体上的 json serializer，需自行序列化
	if err := db.Model(&flowentity.Procdef{}).Where("id = 1").Update("trigger_policy", jsonx.ToStr(custom)).Error; err != nil {
		t.Fatalf("overwrite with a custom policy failed: %v", err)
	}

	if err := migrate(db); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}
	var stored flowentity.Procdef
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if stored.TriggerPolicy == nil || len(stored.TriggerPolicy.Checks) != 0 || stored.TriggerPolicy.DefaultSeverity != flowentity.SeverityDisabled {
		t.Fatalf("an already configured policy must survive a repeated migration, got %+v", stored.TriggerPolicy)
	}
}

// 全新库由 AutoMigrate 直接建出 trigger_policy 列，旧列不存在时迁移不得报错
func TestFlowTriggerPolicyMigrationOnFreshSchema(t *testing.T) {
	db := openFlowMigDB(t)
	if err := db.AutoMigrate(&flowentity.Procdef{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	if err := triggerPolicyMigration(t)(db); err != nil {
		t.Fatalf("the migration must be safe on a schema without the legacy column: %v", err)
	}
}

// 触发策略经实体上的 json serializer 落库后，列里必须是 JSON 文本，回读必须结构一致。
// 这条守卫盯住「列类型/serializer 配错导致策略静默存成空值」这类只在真实读写时暴露的问题
func TestProcdefTriggerPolicySerializer(t *testing.T) {
	db := openFlowMigDB(t)
	if err := db.AutoMigrate(&flowentity.Procdef{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	now := time.Now()
	record := &flowentity.Procdef{Name: "serializer flow", DefKey: "serializer_key", Status: flowentity.ProcdefStatusEnable}
	record.Id = 7
	record.CreateTime = &now
	record.UpdateTime = &now
	record.Creator = "admin"
	record.Modifier = "admin"
	record.TriggerPolicy = &flowentity.TriggerPolicy{
		Version:         flowentity.TriggerPolicyVersion,
		DefaultSeverity: flowentity.SeverityDisabled,
		Checks:          []*flowentity.CheckConfig{{Key: "redis.dangerous-cmd", BizType: "redis_run_cmd_flow", Severity: flowentity.SeverityForbidden}},
		Customs: []*flowentity.CustomCondition{{
			BizType: "db_sql_exec_flow", Severity: flowentity.SeverityRequired,
			When: &flowentity.RuleNode{Kind: flowentity.NodeKindCondition, Field: "tableCount", Op: "gt", Value: 3},
		}},
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}

	var raw string
	if err := db.Table("t_flow_procdef").Select("trigger_policy").Where("id = 7").Scan(&raw).Error; err != nil {
		t.Fatalf("read the raw column failed: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		t.Fatalf("the policy column must store json text, got %q", raw)
	}

	var reloaded flowentity.Procdef
	if err := db.First(&reloaded, 7).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	got := reloaded.TriggerPolicy
	if got == nil || len(got.Checks) != 1 || len(got.Customs) != 1 {
		t.Fatalf("the policy must round-trip, got %+v", got)
	}
	if got.Checks[0].Severity != flowentity.SeverityForbidden || got.Customs[0].When.Value == nil {
		t.Fatalf("nested values must survive the round-trip, got %+v", got)
	}
}
