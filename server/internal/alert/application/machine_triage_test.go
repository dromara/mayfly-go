package application

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	machineapp "mayfly-go/internal/machine/application"
)

// fakeEvaluator 只用来驱动分诊的判定分支，不参与真实指标取值
type fakeEvaluator struct {
	result *service.EvalResult
	err    error
}

func (f *fakeEvaluator) ResourceType() int8                   { return 1 }
func (f *fakeEvaluator) Metrics() []service.MetricDefinition  { return nil }
func (f *fakeEvaluator) ResourceName(uint64) string           { return "" }
func (f *fakeEvaluator) ResourceExists(uint64) bool           { return true }
func (f *fakeEvaluator) ResourceIdsByCodes([]string) []uint64 { return nil }
func (f *fakeEvaluator) Evaluate(context.Context, *service.EvalParam) (*service.EvalResult, error) {
	return f.result, f.err
}

func newRow() *machineapp.MachineHealth {
	return &machineapp.MachineHealth{MachineId: 1, Status: 1, Priority: machineapp.NoHealthHit, Triaged: true}
}

func diskRule(id uint64, name string, priority entity.AlertPriority) *entity.AlertRule {
	rule := &entity.AlertRule{
		Name:     name,
		Priority: priority,
		Condition: &entity.AlertCondition{
			Operator: "and",
			Items:    []entity.ConditionItem{{Metric: "disk_usage", Compare: "gt", Value: 85}},
		},
	}
	rule.Id = id
	return rule
}

func TestEvaluateRuleOnRecordsWorstPriorityAcrossRules(t *testing.T) {
	row := newRow()

	// P2 规则先命中
	evaluateRuleOn(context.Background(), diskRule(10, "磁盘一般告警", entity.AlertPriorityMedium), 1,
		&fakeEvaluator{result: &service.EvalResult{Evaluated: true, Triggered: true, Items: []service.ConditionEval{
			{Metric: "disk_usage", Compare: "gt", Threshold: 85, CurrentValue: 91.5, Satisfied: true},
		}}}, row)
	if row.Priority != int8(entity.AlertPriorityMedium) {
		t.Fatalf("priority = %d, want %d", row.Priority, entity.AlertPriorityMedium)
	}

	// 更严重的 P0 规则也命中：优先级必须取最差（数值更小）
	evaluateRuleOn(context.Background(), diskRule(11, "磁盘紧急告警", entity.AlertPriorityCritical), 1,
		&fakeEvaluator{result: &service.EvalResult{Evaluated: true, Triggered: true, Items: []service.ConditionEval{
			{Metric: "disk_usage", Compare: "gt", Threshold: 85, CurrentValue: 96, Satisfied: true},
		}}}, row)
	if row.Priority != int8(entity.AlertPriorityCritical) {
		t.Fatalf("worst priority must win: got %d, want %d", row.Priority, entity.AlertPriorityCritical)
	}

	// 两条规则都要能下钻到「为什么红」：命中明细逐条保留，且带规则名与阈值
	if len(row.Hits) != 2 {
		t.Fatalf("hits = %d, want 2（每条命中规则都要可解释）", len(row.Hits))
	}
	if row.Hits[1].RuleName != "磁盘紧急告警" || row.Hits[1].Threshold != 85 || row.Hits[1].Current != 96 {
		t.Fatalf("hit detail wrong: %+v", row.Hits[1])
	}
	if !row.Triaged {
		t.Fatal("已得出触发结论时 Triaged 必须为 true")
	}
}

// 取不到当前值时必须标「未判定」：按未越线展示会让用户以为机器正常
func TestEvaluateRuleOnFailClosedWhenUnreliable(t *testing.T) {
	row := newRow()
	evaluateRuleOn(context.Background(), diskRule(10, "r", entity.AlertPriorityHigh), 1,
		&fakeEvaluator{result: &service.EvalResult{Evaluated: false, Triggered: false}}, row)

	if row.Triaged {
		t.Fatal("Evaluated=false must mark the row as not triaged")
	}
	if row.Priority != machineapp.NoHealthHit || len(row.Hits) != 0 {
		t.Fatalf("不可靠判定不得产出命中: priority=%d hits=%d", row.Priority, len(row.Hits))
	}
}

func TestEvaluateRuleOnSwallowsPerMachineError(t *testing.T) {
	row := newRow()
	evaluateRuleOn(context.Background(), diskRule(10, "r", entity.AlertPriorityHigh), 1,
		&fakeEvaluator{err: errors.New("boom")}, row)

	if row.Triaged {
		t.Fatal("evaluate error must mark the row as not triaged")
	}
	if len(row.Hits) != 0 {
		t.Fatal("evaluate error must not produce hits")
	}
}

func TestEvaluateRuleOnNotTriggeredKeepsNormal(t *testing.T) {
	row := newRow()
	evaluateRuleOn(context.Background(), diskRule(10, "r", entity.AlertPriorityHigh), 1,
		&fakeEvaluator{result: &service.EvalResult{Evaluated: true, Triggered: false, Items: []service.ConditionEval{
			{Metric: "disk_usage", Compare: "gt", Threshold: 85, CurrentValue: 20, Satisfied: false},
		}}}, row)

	if !row.Triaged || row.Priority != machineapp.NoHealthHit || len(row.Hits) != 0 {
		t.Fatalf("未越线应保持正常态: triaged=%v priority=%d hits=%d", row.Triaged, row.Priority, len(row.Hits))
	}
}

// 只有「满足」的条件项才进明细，未满足项不得混入（否则提示会指向一个没越线的指标）
func TestApplyHitsOnlyIncludesSatisfiedItems(t *testing.T) {
	row := newRow()
	applyHits(row, diskRule(20, "cpu 或磁盘", entity.AlertPriorityLow), &service.EvalResult{
		Evaluated: true, Triggered: true,
		Items: []service.ConditionEval{
			{Metric: "cpu_rate", Compare: "gt", Threshold: 80, CurrentValue: 30, Satisfied: false},
			{Metric: "mem_rate", Compare: "gt", Threshold: 90, CurrentValue: 97, Satisfied: true},
		},
	})

	if len(row.Hits) != 1 || row.Hits[0].Metric != "mem_rate" {
		t.Fatalf("hits = %+v, want only the satisfied mem_rate item", row.Hits)
	}
	if row.Hits[0].RuleId != 20 || row.Hits[0].Priority != int8(entity.AlertPriorityLow) {
		t.Fatalf("hit must carry its owning rule: %+v", row.Hits[0])
	}
}

func TestMarkUntaggedCoversEveryRow(t *testing.T) {
	rows := []*machineapp.MachineHealth{newRow(), newRow()}
	markUntagged(rows)
	for i, r := range rows {
		if r.Triaged {
			t.Fatalf("row[%d] must be marked not triaged", i)
		}
	}
}
