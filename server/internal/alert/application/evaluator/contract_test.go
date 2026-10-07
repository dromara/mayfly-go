package evaluator

import (
	"testing"

	"mayfly-go/internal/alert/domain/service"
)

// 评估器通用契约：新增评估器（如 Redis/DB）接入时，必须为其实例追加一个
// TestEvaluatorContract_Xxx 用例调用 assertEvaluatorContract 并保证全绿。
// 元信息完整性断言委托给 service.ValidateMetrics 单一真源（与启动自检同一实现）

// assertEvaluatorContract 校验评估器接入的元信息契约
func assertEvaluatorContract(t *testing.T, e service.AlertEvaluator) {
	t.Helper()

	if e.ResourceType() == 0 {
		t.Error("ResourceType() must not return 0 (consts.ResourceType* are non-zero)")
	}
	if err := service.ValidateMetrics(e.Metrics()); err != nil {
		t.Errorf("metric definitions contract violated: %v", err)
	}
}

// TestEvaluatorContract_Machine 机器评估器接入通用契约
func TestEvaluatorContract_Machine(t *testing.T) {
	assertEvaluatorContract(t, &MachineEvaluator{})
}

// Evaluate 语义契约 checklist —— 依赖各评估器自身数据源，无法通用断言，
// 由各评估器自身测试覆盖。新增评估器时逐条核对：
//
//  1. EvalResult.Items 与 rule.Condition.Items 下标严格对齐：指标无法取值时
//     记录 ConditionEval.Err 而非跳过，否则 AND/OR 语义与 Duration 计算因下标错位失真
//  2. 任一条件无法评估时 Evaluated=false（fail-closed）：调用方据此跳过状态迁移，
//     防止把"取不到数据"误判为"已恢复"
//  3. "能否取到数据"类存活指标（如机器 status）在数据源失败时必须给出离线判定
//     而非错误；其余指标在数据源失败时必须返回错误
//  4. 规则条件为空时返回 Evaluated=true 且不触发
//  5. 并发安全：评估器为单例，引擎最多 10 个规则并发调用 Evaluate/Metrics，
//     懒初始化必须用 sync.Once 等手段收敛（参照 MachineEvaluator.metricsOnce）
