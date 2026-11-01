package evaluator

// 数据库实例评估器指标提取的回归测试。
//
// 核心要钉住的是「观测缺口不得被当成健康」：方言不支持会话内省、或实例根本连不上时，
// 依赖会话的指标必须返回错误（让引擎把本次判定标为不可靠），而不是回 0——
// 回 0 会让告警面板显示「活动会话 0、等锁 0」，运维据此认为实例无事发生。

import (
	"errors"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
)

func healthWith(sessions ...*dbi.SessionInfo) *dbi.DbHealth {
	// BaselineExists 置真：这个助手代表「一切都可观测」的健康样本，
	// 否则漂移指标会在「指标注册表一致性」用例里被当成故障
	//
	// TLSCertDaysLeft 同理：证书指标在「没配证书」时按设计返回错误（不可判定），
	// 健康样本必须是「配了证书且还有效」，否则会把合法语义误判成回归
	certDays := 90
	// RecoveryPrepared 同理：PITR 指标在「方言不支持探测」时按设计返回错误，
	// 健康样本必须是「探测过且已就绪」
	pitrReady := true
	return &dbi.DbHealth{InstanceId: 1, Sessions: sessions, SessionsAvailable: true, BaselineExists: true, TLSCertDaysLeft: &certDays, RecoveryPrepared: &pitrReady}
}

// TestDbExtractTLSCertDaysLeft 证书剩余天数的三种取值语义：
// 有证书给天数、没证书返回错误（不是 0）、已过期给负数。
func TestDbExtractTLSCertDaysLeft(t *testing.T) {
	days := 12
	if v, err := extractDbTLSCertDaysLeft(&dbi.DbHealth{TLSCertDaysLeft: &days}, nil); err != nil || v != 12 {
		t.Fatalf("configured cert must yield its remaining days, got v=%v err=%v", v, err)
	}

	expired := -3
	if v, err := extractDbTLSCertDaysLeft(&dbi.DbHealth{TLSCertDaysLeft: &expired}, nil); err != nil || v != -3 {
		t.Fatalf("expired cert must yield a negative day count, got v=%v err=%v", v, err)
	}

	// 没配证书时返回错误而不是 0：0 会被规则读成「今天到期」
	if v, err := extractDbTLSCertDaysLeft(&dbi.DbHealth{}, nil); err == nil {
		t.Fatalf("instance without certificate must error instead of reporting %v days", v)
	}
	if _, err := extractDbTLSCertDaysLeft(nil, errors.New("instance not found")); err == nil {
		t.Fatal("sampling failure must surface as an error")
	}
}

func TestDbExtractPingFail(t *testing.T) {
	if v, err := extractDbPingFail(&dbi.DbHealth{PingErr: errors.New("connection refused")}, nil); err != nil || v != 1 {
		t.Fatalf("unreachable instance must be 1 (offline-like), got v=%v err=%v", v, err)
	}
	if v, err := extractDbPingFail(&dbi.DbHealth{}, nil); err != nil || v != 0 {
		t.Fatalf("reachable instance must be 0, got v=%v err=%v", v, err)
	}
	// 采样本身失败也按不可达给结论：存活类指标不能因为取不到数据而「无值」
	if v, err := extractDbPingFail(nil, errors.New("instance not found")); err != nil || v != 1 {
		t.Fatalf("sampling failure must still yield a reachability verdict, got v=%v err=%v", v, err)
	}
}

// 会话类指标在三种「看不到」的情况下都必须报错：能力缺失、实例不可达、采样失败
func TestDbSessionMetricsRefuseToReportZero(t *testing.T) {
	cases := []struct {
		name   string
		health *dbi.DbHealth
		hErr   error
	}{
		{"dialect without session introspection", &dbi.DbHealth{}, nil},
		{"unreachable instance", &dbi.DbHealth{PingErr: errors.New("no route to host")}, nil},
		{"sampling failed", nil, errors.New("instance not found")},
	}
	extracts := []struct {
		name string
		fn   func(*dbi.DbHealth, error) (float64, error)
	}{
		{"sessions_active", extractDbSessionsActive},
		{"session_max_elapsed", extractDbSessionMaxElapsed},
		{"lock_wait_count", extractDbLockWaitCount},
	}

	for _, c := range cases {
		for _, ex := range extracts {
			v, err := ex.fn(c.health, c.hErr)
			if err == nil {
				t.Errorf("%s must not report %v when %s: session metrics would look healthy while nothing is observable", ex.name, v, c.name)
			}
		}
	}
}

func TestDbSessionMetricsValues(t *testing.T) {
	health := healthWith(
		&dbi.SessionInfo{Id: "1", ElapsedMs: 1500},
		&dbi.SessionInfo{Id: "2", ElapsedMs: 42000, LockWait: true},
		&dbi.SessionInfo{Id: "3", LockWait: true},
	)

	if v, err := extractDbSessionsActive(health, nil); err != nil || v != 3 {
		t.Fatalf("sessions_active want 3, got %v err %v", v, err)
	}
	// 42000ms → 42s：秒数由指标给出，阈值（多少算慢）归规则表达
	if v, err := extractDbSessionMaxElapsed(health, nil); err != nil || v != 42 {
		t.Fatalf("session_max_elapsed want 42, got %v err %v", v, err)
	}
	if v, err := extractDbLockWaitCount(health, nil); err != nil || v != 2 {
		t.Fatalf("lock_wait_count want 2, got %v err %v", v, err)
	}

	// 可观测但确实没有会话：必须是 0 且不报错（与上面的「看不到」区分开）
	empty, err := extractDbSessionsActive(healthWith(), nil)
	if err != nil || empty != 0 {
		t.Fatalf("observable but idle instance must report 0 without error, got %v err %v", empty, err)
	}
}

// TestDbSchemaDriftMetric 漂移表数指标：没建基线必须是「无从判断」而不是 0。
//
// 报 0 会被规则当成「结构很稳定」并在界面上标绿，而事实是根本还没配基线——
// 把一个缺口当成好消息正是本模块最不能接受失效方式
func TestDbSchemaDriftMetric(t *testing.T) {
	if _, err := extractDbSchemaDrift(&dbi.DbHealth{InstanceId: 1}, nil); err == nil {
		t.Fatal("no baseline must not report 0 drifted tables")
	}
	if _, err := extractDbSchemaDrift(nil, errors.New("instance gone")); err == nil {
		t.Fatal("unavailable health must error")
	}

	stable := &dbi.DbHealth{BaselineExists: true, DriftedTables: 0}
	if v, err := extractDbSchemaDrift(stable, nil); err != nil || v != 0 {
		t.Fatalf("baseline with nothing changed must report 0, got v=%v err=%v", v, err)
	}
	drifted := &dbi.DbHealth{BaselineExists: true, DriftedTables: 3}
	if v, err := extractDbSchemaDrift(drifted, nil); err != nil || v != 3 {
		t.Fatalf("want 3 drifted tables, got v=%v err=%v", v, err)
	}
}

// TestDbExtractPitrNotReady PITR 前提指标的三种取值语义：
// 未就绪给 1、就绪给 0、方言不支持探测返回错误（不是 0——0 会被读成「已就绪」并标绿）。
func TestDbExtractPitrNotReady(t *testing.T) {
	notReady, ready := false, true
	if v, err := extractDbPitrNotReady(&dbi.DbHealth{RecoveryPrepared: &notReady}, nil); err != nil || v != 1 {
		t.Fatalf("not ready must be 1, got v=%v err=%v", v, err)
	}
	if v, err := extractDbPitrNotReady(&dbi.DbHealth{RecoveryPrepared: &ready}, nil); err != nil || v != 0 {
		t.Fatalf("ready must be 0, got v=%v err=%v", v, err)
	}
	if _, err := extractDbPitrNotReady(&dbi.DbHealth{}, nil); err == nil {
		t.Fatal("unobservable recovery readiness must error, not report 0")
	}
	if _, err := extractDbPitrNotReady(nil, errors.New("instance gone")); err == nil {
		t.Fatal("health error must propagate")
	}
}

// TestDbEvaluatorContractMetricsRegistered 指标定义与注册表一致性：
// Metrics() 暴露的每个 key 都必须能被 extract 取到值或给出明确错误，不能出现「下拉里有、取值必 panic」
func TestDbEvaluatorContractMetricsRegistered(t *testing.T) {
	e := &DbEvaluator{}
	e.ensureMetrics()

	health := healthWith(&dbi.SessionInfo{Id: "1"})
	for _, def := range e.Metrics() {
		if _, err := e.metrics.extract(health, nil, def.Key); err != nil {
			t.Errorf("metric %s extract on a healthy sample must not fail: %v", def.Key, err)
		}
	}

	unknown, err := e.metrics.extract(health, nil, "metric_never_registered")
	if err == nil {
		t.Errorf("unknown metric must error, got value %v", unknown)
	}
}
