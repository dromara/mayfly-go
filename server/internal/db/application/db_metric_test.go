package application

// 指标采样口径：三条「不可观测就不写」的红线。
// 写成 0 会被图表读成事实（0 条连接 / 今天到期 / 结构很稳定），比缺一个点危险得多。

import (
	"errors"
	"testing"
	"time"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

func metricValues(samples []*entity.DbMetricSample) map[string]float64 {
	values := make(map[string]float64, len(samples))
	for _, sample := range samples {
		values[sample.Metric] = sample.Value
	}
	return values
}

func TestMetricSamplesFromHealth(t *testing.T) {
	at := time.Now()

	// 1) 可达 + 会话可内省 + 有证书 + 有基线：六项都写
	days := 30
	health := &dbi.DbHealth{
		SessionsAvailable: true,
		Sessions: []*dbi.SessionInfo{
			{Id: "1", ElapsedMs: 2000},
			{Id: "2", ElapsedMs: 9000, LockWait: true},
			{Id: "3", ElapsedMs: 1000, LockWait: true},
		},
		TLSCertDaysLeft: &days,
		BaselineExists:  true,
		DriftedTables:   2,
	}
	values := metricValues(metricSamplesFromHealth(7, health, at))
	expect := map[string]float64{
		MetricPingFail:          0,
		MetricSessionsActive:    3,
		MetricSessionMaxElapsed: 9,
		MetricLockWaitCount:     2,
		MetricTLSCertDays:       30,
		MetricSchemaDrift:       2,
	}
	for metric, want := range expect {
		if got, ok := values[metric]; !ok || got != want {
			t.Fatalf("%s = %v (present=%v), want %v", metric, got, ok, want)
		}
	}

	// 2) 会话不可内省：不写连接数/最长耗时/等锁数（写 0 会被读成「没有连接」）
	values = metricValues(metricSamplesFromHealth(7, &dbi.DbHealth{PingErr: errors.New("refused")}, at))
	if values[MetricPingFail] != 1 {
		t.Fatalf("unreachable instance must record ping_fail=1, got %v", values[MetricPingFail])
	}
	for _, metric := range []string{MetricSessionsActive, MetricSessionMaxElapsed, MetricLockWaitCount} {
		if _, ok := values[metric]; ok {
			t.Fatalf("%s must be absent when sessions are not introspectable", metric)
		}
	}

	// 3) PITR 前提：探测过就写（1 就绪 / 0 未就绪），没探测到不写
	pitrReady := true
	values = metricValues(metricSamplesFromHealth(7, &dbi.DbHealth{SessionsAvailable: true, RecoveryPrepared: &pitrReady}, at))
	if values[MetricPitrReady] != 1 {
		t.Fatalf("ready instance must record pitr_ready=1, got %v", values[MetricPitrReady])
	}
	pitrNotReady := false
	values = metricValues(metricSamplesFromHealth(7, &dbi.DbHealth{RecoveryPrepared: &pitrNotReady}, at))
	if values[MetricPitrReady] != 0 {
		t.Fatalf("not-ready instance must record pitr_ready=0, got %v", values[MetricPitrReady])
	}
	if _, ok := metricValues(metricSamplesFromHealth(7, &dbi.DbHealth{}, at))[MetricPitrReady]; ok {
		t.Fatal("pitr_ready must be absent when readiness was not probed")
	}

	// 4) 没配证书 / 没建基线：对应指标不写（0 会被读成「今天到期」「结构稳定」）
	values = metricValues(metricSamplesFromHealth(7, &dbi.DbHealth{SessionsAvailable: true}, at))
	if _, ok := values[MetricTLSCertDays]; ok {
		t.Fatal("tls_cert_days must be absent when no certificate is configured")
	}
	if _, ok := values[MetricSchemaDrift]; ok {
		t.Fatal("schema_drift_tables must be absent when no baseline exists")
	}
}
