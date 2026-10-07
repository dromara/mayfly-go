package entity

import "testing"

// TestCheckConfigured 检查项是否被策略显式纳管的判定。
//
// 业务侧靠它区分「管理员没管这件事」与「管理员管了但选了轻处置」：
// 前者要继续由历史硬基线兜住，后者必须以配置为准，两者混成一谈就会让降级操作静默失效
func TestCheckConfigured(t *testing.T) {
	policy := &TriggerPolicy{
		Version: TriggerPolicyVersion,
		Checks: []*CheckConfig{
			{Key: "db.dml-without-where", BizType: "db_sql_exec_flow", Severity: SeverityRequired},
			// 级别为「不处置」也算纳管：这是管理员显式选择的结果，不是没配过
			{Key: "redis.write-cmd", BizType: "redis_run_cmd_flow", Severity: SeverityDisabled},
			nil, // 历史脏数据里的 null 元素不能把判定打成 panic
		},
	}

	if !policy.CheckConfigured("redis_run_cmd_flow", "redis.write-cmd") {
		t.Fatal("a check configured with the disabled severity must still count as governed")
	}
	if !policy.CheckConfigured("db_sql_exec_flow", "db.dml-without-where") {
		t.Fatal("a configured check of the same biz type must be reported")
	}
	// 同一条检查项挂在别的场景上不等于本场景被纳管
	if policy.CheckConfigured("machine_run_cmd_flow", "db.dml-without-where") {
		t.Fatal("a check of another biz type must not be treated as governing this scenario")
	}
	if policy.CheckConfigured("machine_run_cmd_flow", "machine.cmd-blacklisted") {
		t.Fatal("an absent check must be reported as unconfigured so the caller can keep its baseline")
	}

	var nilPolicy *TriggerPolicy
	if nilPolicy.CheckConfigured("machine_run_cmd_flow", "machine.cmd-blacklisted") {
		t.Fatal("a nil policy has configured nothing")
	}
}
