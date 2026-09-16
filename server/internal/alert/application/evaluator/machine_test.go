package evaluator

import (
	"errors"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	"testing"
)

// ==================== compareValue 测试 ====================

func TestCompareValue_Gt(t *testing.T) {
	tests := []struct {
		name      string
		current   float64
		threshold float64
		want      bool
	}{
		{"90 > 80 → true", 90, 80, true},
		{"80 > 80 → false", 80, 80, false},
		{"70 > 80 → false", 70, 80, false},
		{"0.1 > 0 → true", 0.1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValue(tt.current, "gt", tt.threshold)
			if got != tt.want {
				t.Errorf("compareValue(%v, gt, %v) = %v, want %v", tt.current, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestCompareValue_Gte(t *testing.T) {
	tests := []struct {
		name      string
		current   float64
		threshold float64
		want      bool
	}{
		{"90 >= 80 → true", 90, 80, true},
		{"80 >= 80 → true", 80, 80, true},
		{"70 >= 80 → false", 70, 80, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValue(tt.current, "gte", tt.threshold)
			if got != tt.want {
				t.Errorf("compareValue(%v, gte, %v) = %v, want %v", tt.current, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestCompareValue_Lt(t *testing.T) {
	tests := []struct {
		name      string
		current   float64
		threshold float64
		want      bool
	}{
		{"70 < 80 → true", 70, 80, true},
		{"80 < 80 → false", 80, 80, false},
		{"90 < 80 → false", 90, 80, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValue(tt.current, "lt", tt.threshold)
			if got != tt.want {
				t.Errorf("compareValue(%v, lt, %v) = %v, want %v", tt.current, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestCompareValue_Lte(t *testing.T) {
	tests := []struct {
		name      string
		current   float64
		threshold float64
		want      bool
	}{
		{"70 <= 80 → true", 70, 80, true},
		{"80 <= 80 → true", 80, 80, true},
		{"90 <= 80 → false", 90, 80, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValue(tt.current, "lte", tt.threshold)
			if got != tt.want {
				t.Errorf("compareValue(%v, lte, %v) = %v, want %v", tt.current, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestCompareValue_Eq(t *testing.T) {
	tests := []struct {
		name      string
		current   float64
		threshold float64
		want      bool
	}{
		{"80 == 80 → true", 80, 80, true},
		{"80.0 == 80 → true", 80.0, 80, true},
		{"79.9 == 80 → false", 79.9, 80, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValue(tt.current, "eq", tt.threshold)
			if got != tt.want {
				t.Errorf("compareValue(%v, eq, %v) = %v, want %v", tt.current, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestCompareValue_Neq(t *testing.T) {
	tests := []struct {
		name      string
		current   float64
		threshold float64
		want      bool
	}{
		{"80 != 80 → false", 80, 80, false},
		{"79 != 80 → true", 79, 80, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValue(tt.current, "neq", tt.threshold)
			if got != tt.want {
				t.Errorf("compareValue(%v, neq, %v) = %v, want %v", tt.current, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestCompareValue_UnknownOperator(t *testing.T) {
	if compareValue(80, "unknown", 80) {
		t.Error("unknown operator should return false")
	}
}

// ==================== extractMetric 测试（使用 mock stats） ====================

// mockStats 模拟 mcm.Stats 结构体的导出字段
type mockStats struct {
	CPU     mockCPU
	MemInfo mockMemInfo
	FSInfos []mockFSInfo
}

type mockCPU struct {
	Idle float64
}

type mockMemInfo struct {
	Total     uint64
	Available uint64
}

type mockFSInfo struct {
	MountPoint string
	Used       uint64
	Free       uint64
}

func TestExtractCpuRate(t *testing.T) {
	stats := &mockStats{CPU: mockCPU{Idle: 20.0}}
	rate, err := extractCpuRate(stats, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// CPU usage = 100 - idle = 100 - 20 = 80
	if rate != 80.0 {
		t.Errorf("expected cpu rate 80.0, got %v", rate)
	}
}

func TestExtractCpuRate_ZeroIdle(t *testing.T) {
	stats := &mockStats{CPU: mockCPU{Idle: 0.0}}
	rate, err := extractCpuRate(stats, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 100.0 {
		t.Errorf("expected cpu rate 100.0, got %v", rate)
	}
}

func TestExtractMemRate(t *testing.T) {
	// Total=1000, Available=300 → usage = (1000-300)/1000 * 100 = 70%
	stats := &mockStats{MemInfo: mockMemInfo{Total: 1000, Available: 300}}
	rate, err := extractMemRate(stats, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 70.0 {
		t.Errorf("expected mem rate 70.0, got %v", rate)
	}
}

// TestExtractMemRate_ZeroTotal 总内存为 0 说明采集数据不可信，
// 必须返回错误让上层把本轮判定标记为不可靠，而不是伪造成 0% 触发"已恢复"
func TestExtractMemRate_ZeroTotal(t *testing.T) {
	stats := &mockStats{MemInfo: mockMemInfo{Total: 0, Available: 0}}
	if _, err := extractMemRate(stats, nil); err == nil {
		t.Error("expected error when memory total is 0")
	}
}

func TestExtractDiskUsage_RootMount(t *testing.T) {
	// Used=600, Free=400 → usage = 600/(600+400) * 100 = 60%
	stats := &mockStats{FSInfos: []mockFSInfo{
		{MountPoint: "/", Used: 600, Free: 400},
	}}
	rate, err := extractDiskUsage(stats, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 60.0 {
		t.Errorf("expected disk usage 60.0, got %v", rate)
	}
}

// TestExtractDiskUsage_NoRootMount 无根分区时取所有挂载点中最高使用率：
// 告警场景关注的是"最满的盘"，此前恒取根分区会让容器化/独立数据盘机器永不告警
func TestExtractDiskUsage_NoRootMount(t *testing.T) {
	stats := &mockStats{FSInfos: []mockFSInfo{
		{MountPoint: "/data", Used: 600, Free: 400},
		{MountPoint: "/backup", Used: 800, Free: 200},
	}}
	rate, err := extractDiskUsage(stats, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 80.0 {
		t.Errorf("expected max disk usage 80.0, got %v", rate)
	}
}

// TestExtractDiskUsage_PrefersRootMount 存在根分区时以根分区为准
func TestExtractDiskUsage_PrefersRootMount(t *testing.T) {
	stats := &mockStats{FSInfos: []mockFSInfo{
		{MountPoint: "/data", Used: 900, Free: 100},
		{MountPoint: "/", Used: 600, Free: 400},
	}}
	rate, err := extractDiskUsage(stats, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 60.0 {
		t.Errorf("expected root mount usage 60.0, got %v", rate)
	}
}

// TestExtractDiskUsage_EmptyFSInfos 无任何文件系统信息时返回错误（数据不可信）
func TestExtractDiskUsage_EmptyFSInfos(t *testing.T) {
	stats := &mockStats{FSInfos: []mockFSInfo{}}
	if _, err := extractDiskUsage(stats, nil); err == nil {
		t.Error("expected error when filesystem info is empty")
	}
}

// TestExtractDiskUsage_AllZeroTotal 全部分区容量为 0 时返回错误，避免除零得到 NaN
func TestExtractDiskUsage_AllZeroTotal(t *testing.T) {
	stats := &mockStats{FSInfos: []mockFSInfo{{MountPoint: "/", Used: 0, Free: 0}}}
	if _, err := extractDiskUsage(stats, nil); err == nil {
		t.Error("expected error when all partitions have zero size")
	}
}

func TestExtractMetric_UnknownMetric(t *testing.T) {
	stats := &mockStats{}
	e := &MachineEvaluator{}
	e.ensureMetrics()
	_, err := e.extractMetric(stats, nil, "unknown_metric")
	if err == nil {
		t.Error("expected error for unknown metric")
	}
}

func TestExtractMetric_Status(t *testing.T) {
	stats := &mockStats{}
	e := &MachineEvaluator{}
	e.ensureMetrics()
	val, err := e.extractMetric(stats, nil, MetricStatus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 1 {
		t.Errorf("expected status=1 (online), got %v", val)
	}
}

// TestExtractMetric_StatusOffline 取不到运行状态时必须判定为离线(0)，
// 否则宕机检测形同虚设（此前 status 指标恒为 1）
func TestExtractMetric_StatusOffline(t *testing.T) {
	e := &MachineEvaluator{}
	e.ensureMetrics()
	val, err := e.extractMetric(nil, errors.New("machine offline"), MetricStatus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Errorf("expected status=0 (offline), got %v", val)
	}
}

// TestExtractMetric_StatsErrorBlocksNonStatusMetric 采集失败时非 status 指标必须返回错误，
// 让上层把本次判定标记为不可靠，避免误触发与误恢复
func TestExtractMetric_StatsErrorBlocksNonStatusMetric(t *testing.T) {
	e := &MachineEvaluator{}
	e.ensureMetrics()
	for _, metric := range []string{MetricCpuRate, MetricMemRate, MetricDiskUsage} {
		if _, err := e.extractMetric(nil, errors.New("collect error"), metric); err == nil {
			t.Errorf("metric %s: expected error when stats unavailable", metric)
		}
	}
}

// ==================== 判定可靠性测试 ====================

func TestUsesStatusMetric(t *testing.T) {
	e := &MachineEvaluator{}
	e.ensureMetrics()
	if !e.usesStatusMetric(&entity.AlertCondition{Items: []entity.ConditionItem{{Metric: MetricStatus, Compare: "eq", Value: 0}}}) {
		t.Error("condition with status metric must be detected")
	}
	if e.usesStatusMetric(&entity.AlertCondition{Items: []entity.ConditionItem{{Metric: MetricCpuRate}}}) {
		t.Error("condition without status metric must not be detected")
	}
	if e.usesStatusMetric(nil) {
		t.Error("nil condition must not be detected")
	}
}

// TestIsTriggered_IgnoresUnevaluableItem AND 下含不可判定项时整体不触发，
// 由 Evaluated=false 阻止调用方据此做状态迁移
func TestIsTriggered_AndWithUnevaluableItem(t *testing.T) {
	e := &MachineEvaluator{}
	items := []service.ConditionEval{
		{Metric: MetricCpuRate, Satisfied: true},
		{Metric: "ghost_metric", Err: "unknown metric: ghost_metric"},
	}
	if e.isTriggered("and", items) {
		t.Error("AND with an unevaluable item must not trigger")
	}
}

// TestIsTriggered_OrSatisfiedBySingleItem OR 下任一满足即触发，不受另一项取值失败影响
func TestIsTriggered_OrSatisfiedBySingleItem(t *testing.T) {
	e := &MachineEvaluator{}
	items := []service.ConditionEval{
		{Metric: "ghost_metric", Err: "unknown metric: ghost_metric"},
		{Metric: MetricMemRate, Satisfied: true},
	}
	if !e.isTriggered("or", items) {
		t.Error("OR with one satisfied item must trigger")
	}
}

// TestHasUnevaluableItem 修复前评估器会直接跳过取不到值的条件项，
// 使 result.Items 与 rule.Condition.Items 下标错位，Duration 因此取错条件项
func TestHasUnevaluableItem(t *testing.T) {
	if !hasUnevaluableItem([]service.ConditionEval{{Metric: MetricCpuRate}, {Metric: "x", Err: "boom"}}) {
		t.Error("item with Err must be reported as unevaluable")
	}
	if hasUnevaluableItem([]service.ConditionEval{{Metric: MetricCpuRate, Satisfied: true}}) {
		t.Error("all-evaluable items must not be reported")
	}
}
