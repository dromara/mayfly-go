package migrations

import (
	"strings"
	"testing"

	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func flowConditionMigration(t *testing.T) func(*gorm.DB) error {
	t.Helper()
	for _, m := range V1_12() {
		if m.ID == "v1.12.0-flow-condition-tree" {
			return m.Migrate
		}
	}
	t.Fatal("flow condition tree migration not found")
	return nil
}

// 旧写法只能翻译成等价的条件下，不能整条丢掉：
// 丢掉会签条件等于让「第一个审批人通过即完成」，那是把审批要求静默撤掉
func TestLegacyConditionsTranslateToEquivalentRules(t *testing.T) {
	cases := []struct {
		name     string
		template string
		want     map[string]any
	}{
		{"或签", `{{ eq .nrOfCompleted 1.0 }}`, map[string]any{"kind": "condition", "field": "nrOfCompleted", "op": "gte", "value": 1.0}},
		{"会签", `{{ eq .nrOfAll .nrOfCompleted }}`, map[string]any{"kind": "condition", "field": "nrOfCompletedRate", "op": "gte", "value": 1.0}},
		// 旧设计器能产出的写法只有两种，多空格与换行归一后仍要能命中
		{"或签-多余空格", `{{  eq   .nrOfCompleted 1.0  }}`, map[string]any{"kind": "condition", "field": "nrOfCompleted", "op": "gte", "value": 1.0}},
	}

	for _, tc := range cases {
		graph := `{"nodes":[{"key":"a","type":"usertask","extra":{"completionCondition":"` + strings.ReplaceAll(tc.template, `"`, `\"`) + `"}}]}`
		updated, changed, untranslated, migrateErr := translateFlowConditions(graph)
		if migrateErr != nil || !changed {
			t.Fatalf("%s: the flow def must be rewritten, got changed=%v err=%v", tc.name, changed, migrateErr)
		}
		if len(untranslated) > 0 {
			t.Fatalf("%s: a known legacy form must translate without leftovers, got %v", tc.name, untranslated)
		}

		parsed, err := jsonx.ToMapByStr(updated)
		if err != nil {
			t.Fatalf("%s: the translated flow def must stay valid json: %v", tc.name, err)
		}
		node := parsed["nodes"].([]any)[0].(map[string]any)
		if got := node["extra"].(map[string]any)["completionCondition"]; !collxDeepEqual(got, tc.want) {
			t.Fatalf("%s: translated into %#v, want %#v", tc.name, got, tc.want)
		}
		if strings.Contains(updated, "{{") {
			t.Fatalf("%s: the go template must not survive the translation, got %s", tc.name, updated)
		}
	}
}

// 认不出的写法必须原样保留并交给人重新配置：
// 按包含匹配猜一个条件会把「至少 3 人通过」悄悄放宽成「1 人通过」
func TestUntranslatableConditionIsLeftUntouched(t *testing.T) {
	graph := `{"nodes":[{"key":"a","type":"usertask","extra":{"completionCondition":"{{ if gt .nrOfCompleted 2.0 }}1{{ end }}"}}]}`
	updated, changed, _, err := translateFlowConditions(graph)
	if err != nil {
		t.Fatalf("an untranslatable condition must not fail the migration, got %v", err)
	}
	if changed || updated != graph {
		t.Fatalf("the original text must survive so the runtime reports it instead of guessing, changed=%v content=%s", changed, updated)
	}
}

// 已经是条件树的内容必须原样保留：重复执行迁移不能把新数据再翻译一遍
func TestTranslatedConditionsAreLeftAlone(t *testing.T) {
	graph := `{"nodes":[{"key":"a","type":"usertask","extra":{"completionCondition":{"kind":"condition","field":"nrOfCompleted","op":"gte","value":1}}}]}`
	updated, changed, _, err := translateFlowConditions(graph)
	if err != nil {
		t.Fatalf("a modern flow def must not fail the migration, got %v", err)
	}
	if changed || updated != graph {
		t.Fatalf("an already translated condition must be left untouched, changed=%v content=%s", changed, updated)
	}

	// 空条件（老数据里连完成条件都没填）直接删掉键，交给运行期按默认语义处理
	empty := `{"nodes":[{"key":"a","type":"usertask","extra":{"completionCondition":"   "}}]}`
	updated, changed, _, err = translateFlowConditions(empty)
	if err != nil || !changed {
		t.Fatalf("a blank condition must be cleaned up, got changed=%v err=%v", changed, err)
	}
	if strings.Contains(updated, "completionCondition") {
		t.Fatalf("the blank condition key must be removed, got %s", updated)
	}
}

// 真实落库一轮：迁移要能读旧行并写回，且第二次执行不再改动
func TestFlowConditionMigrationOnDatabase(t *testing.T) {
	db := openFlowMigDB(t)
	if err := db.AutoMigrate(&flowentity.Procdef{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	now := time.Now()
	record := &flowentity.Procdef{
		Name: "condition flow", DefKey: "condition_key", Status: flowentity.ProcdefStatusEnable,
		FlowDef: `{"nodes":[{"name":"审批","key":"n1","type":"usertask","extra":{"candidates":["1"],"completionCondition":"{{ eq .nrOfAll .nrOfCompleted }}"}}],` +
			`"edges":[{"name":"通过","key":"e1","sourceNodeKey":"n1","targetNodeKey":"n2","extra":{"condition":"{{ eq .nrOfCompleted 1.0 }}"}}]}`,
	}
	record.Id = 1
	record.CreateTime = &now
	record.UpdateTime = &now
	record.Creator = "admin"
	record.Modifier = "admin"
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	migrate := flowConditionMigration(t)
	if err := migrate(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	var stored flowentity.Procdef
	if err := db.First(&stored, 1).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	flowDef := stored.GetFlowDef()
	if flowDef == nil {
		t.Fatalf("the migrated flow def must stay readable, got %s", stored.FlowDef)
	}

	edge := flowDef.Edges[0]
	condition, err := edge.Condition()
	if err != nil {
		t.Fatalf("the migrated edge condition must parse as a rule node: %v", err)
	}
	if condition == nil || condition.Field != "nrOfCompleted" || condition.Op != "gte" {
		t.Fatalf("the edge condition must be translated into the or-sign rule, got %+v", condition)
	}

	usertask := flowDef.Nodes[0]
	completion, err := flowentity.RuleNodeFromExtra(usertask.Extra, flowentity.FlowNodeCompletionConditionKey)
	if err != nil {
		t.Fatalf("the migrated completion condition must parse as a rule node: %v", err)
	}
	if completion == nil || completion.Field != "nrOfCompletedRate" {
		t.Fatalf("the completion condition must be translated into the and-sign rule, got %+v", completion)
	}
	if strings.Contains(stored.FlowDef, "{{") {
		t.Fatalf("no go template may remain in the flow def, got %s", stored.FlowDef)
	}

	// 第二次执行必须什么都不改：条件已是对象，翻译函数只处理字符串形态
	before := stored.FlowDef
	if err := migrate(db); err != nil {
		t.Fatalf("the migration must be idempotent, got %v", err)
	}
	if err := db.First(&stored, 1).Error; err != nil {
		t.Fatalf("reload after the second run failed: %v", err)
	}
	if stored.FlowDef != before {
		t.Fatalf("a repeated migration rewrote the flow def:\n before=%s\n after=%s", before, stored.FlowDef)
	}
}

// 条件组与策略变更两张表必须由迁移建出来，且重复执行安全
func TestFlowSegmentAndPolicyHistoryTables(t *testing.T) {
	db := openFlowMigDB(t)
	for _, m := range V1_12() {
		if m.ID == "v1.12.0-flow-rule-segment-policy-history-tables" {
			if err := m.Migrate(db); err != nil {
				t.Fatalf("the table migration failed: %v", err)
			}
			if err := m.Migrate(db); err != nil {
				t.Fatalf("the table migration must be idempotent, got %v", err)
			}
			break
		}
	}

	migrator := db.Migrator()
	for _, table := range []any{new(flowentity.RuleSegment), new(flowentity.ProcdefPolicyHis)} {
		if !migrator.HasTable(table) {
			t.Fatalf("the table for %T must exist after the migration", table)
		}
	}

	// 条件组的条件树走 serializer:json，落库必须是 JSON 文本且能原样读回
	now := time.Now()
	segment := &flowentity.RuleSegment{Ref: "dba_members", Name: "DBA 成员", BizType: "db_sql_exec_flow"}
	segment.CreateTime = &now
	segment.UpdateTime = &now
	segment.Creator = "admin"
	segment.CreatorId = 1
	segment.Modifier = "admin"
	segment.ModifierId = 1
	segment.RuleNode = &flowentity.RuleNode{Kind: flowentity.NodeKindCondition, Field: "sql", Op: "contains", Value: "orders"}
	if err := db.Create(segment).Error; err != nil {
		t.Fatalf("create condition group failed: %v", err)
	}
	var raw string
	if err := db.Table("t_flow_rule_segment").Select("rule_node").Where("id = ?", segment.Id).Scan(&raw).Error; err != nil {
		t.Fatalf("read the raw rule node column failed: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		t.Fatalf("the rule node column must store json text, got %q", raw)
	}
	var reloaded flowentity.RuleSegment
	if err := db.First(&reloaded, segment.Id).Error; err != nil {
		t.Fatalf("reload condition group failed: %v", err)
	}
	if reloaded.RuleNode == nil || reloaded.RuleNode.Field != "sql" {
		t.Fatalf("the rule node must round trip, got %+v", reloaded.RuleNode)
	}
}

// collxDeepEqual 比较翻译结果与期望结构。
// 结构里既有字符串又有 float64，用反射比较会因指针类型差异误判，这里按 JSON 文本比对更稳
func collxDeepEqual(got any, want map[string]any) bool {
	return jsonx.ToStr(collx.M{"v": got}) == jsonx.ToStr(collx.M{"v": want})
}

// 条件组标识不能带 DB 唯一索引：该表软删除，唯一索引会把已删除的行一起算进来，
// 删掉一个条件组后同名标识就再也建不出来
func TestRuleSegmentRefAllowsReuseAfterSoftDelete(t *testing.T) {
	db := openFlowMigDB(t)
	for _, m := range V1_12() {
		if m.ID == "v1.12.0-flow-rule-segment-policy-history-tables" || m.ID == "v1.12.0-flow-rule-segment-ref-drop-unique-index" {
			require.NoError(t, m.Migrate(db))
		}
	}

	now := time.Now()
	seed := func(name string) *flowentity.RuleSegment {
		segment := &flowentity.RuleSegment{Ref: "dba_members", Name: name, BizType: "db_sql_exec_flow"}
		segment.CreateTime = &now
		segment.UpdateTime = &now
		segment.Creator = "admin"
		segment.Modifier = "admin"
		return segment
	}

	first := seed("DBA 成员")
	require.NoError(t, db.Create(first).Error)
	require.NoError(t, db.Delete(first).Error)
	// 软删除后同名重建必须成功，这正是唯一索引会挡住的场景
	require.NoError(t, db.Create(seed("DBA 成员（新）")).Error)

	for _, index := range indexesOf(t, db) {
		require.NotEqual(t, "uk_flow_segment_ref", index, "旧的唯一索引必须由迁移删除掉")
	}
}

func indexesOf(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	indexes, err := db.Migrator().GetIndexes(new(flowentity.RuleSegment))
	require.NoError(t, err)
	names := make([]string, 0, len(indexes))
	for _, index := range indexes {
		names = append(names, index.Name())
	}
	return names
}
