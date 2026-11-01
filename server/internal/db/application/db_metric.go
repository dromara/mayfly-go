package application

// 实例指标采样与趋势查询。
//
// 采样口径与告警评估器同源（都取 instanceApp.Health 的事实），但**存储与保留期独立**：
// 评估器按规则需要现算，图表按固定周期落点。两条链路各自演进，不会因为改了图表保留期
// 就悄悄改变告警的判定依据。
//
// 写入策略：一轮采样只写一条批量 INSERT；某个实例取不到会话时只记 ping_fail，
// 不写「连接数 0」——0 会被图表读成「这台库没有连接」，与「看不到」是两件事。

import (
	"context"
	"sort"
	"time"

	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/taskx"

	"mayfly-go/internal/db/domain/repository"
)

// 指标名：与会话/锁/证书的事实一一对应
const (
	MetricSessionsActive    = "sessions_active"
	MetricSessionMaxElapsed = "session_max_elapsed"
	MetricLockWaitCount     = "lock_wait_count"
	MetricPingFail          = "ping_fail"
	MetricTLSCertDays       = "tls_cert_days"
	MetricSchemaDrift       = "schema_drift_tables"
	MetricPitrReady         = "pitr_ready"
)

// 采样周期与保留期的默认值。
//
// 5 分钟：多数「连接数涨上来了」的排查看的是小时级曲线，更密只增加写放大；
// 7 天：覆盖一个完整的「上周五到现在」对比，再长就该上专业监控了。
// 两者都可由 DbmsConfig 覆盖——大部署（几百实例）每轮采样都要连一遍库，
// 密度直接决定对被管库的压力，必须让运维能按规模调。
const (
	defaultMetricSampleMinutes = 5
	defaultMetricRetainDays    = 7
	// metricTickSeconds 调度器的最小节拍：配置变更无需重启，靠节拍内判断是否到期
	metricTickSeconds = 60
)

// metricSampleInterval 当前生效的采样周期（配置 <=0 时用默认值）
func metricSampleInterval() time.Duration {
	minutes := defaultMetricSampleMinutes
	if dbms := config.GetDbms(); dbms != nil && dbms.MetricSampleMinutes > 0 {
		minutes = dbms.MetricSampleMinutes
	}
	return time.Duration(minutes) * time.Minute
}

// metricRetention 当前生效的保留期（配置 <=0 时用默认值）
func metricRetention() time.Duration {
	days := defaultMetricRetainDays
	if dbms := config.GetDbms(); dbms != nil && dbms.MetricRetainDays > 0 {
		days = dbms.MetricRetainDays
	}
	return time.Duration(days) * 24 * time.Hour
}

// metricSeriesMaxPoints 单次查询返回点数上限：前端画折线，超过这个量只会让浏览器卡
const metricSeriesMaxPoints = 720

// MetricPoint 一个采样点
type MetricPoint struct {
	Ts    int64   `json:"ts"` // 毫秒时间戳
	Value float64 `json:"value"`
}

// MetricSeries 一条趋势
type MetricSeries struct {
	Metric string        `json:"metric"`
	Points []MetricPoint `json:"points"`
}

// DbMetricApp 实例指标采样与查询
type DbMetricApp interface {
	base.App[*entity.DbMetricSample]

	// SampleAll 对所有实例采样一轮，返回写入点数
	SampleAll(ctx context.Context) (int, error)

	// Series 取某实例某指标最近 minutes 分钟的趋势
	Series(ctx context.Context, instanceId uint64, metric string, minutes int) (*MetricSeries, error)

	// TimerSample 注册周期采样任务
	TimerSample()
}

func GetDbMetricApp() DbMetricApp {
	return ioc.Get[DbMetricApp]()
}

type dbMetricAppImpl struct {
	base.AppImpl[*entity.DbMetricSample, repository.DbMetricSample]

	instanceApp Instance `inject:"T"`
}

var _ DbMetricApp = (*dbMetricAppImpl)(nil)

// metricNames 允许查询的指标集合：查询侧必须按白名单校验，
// 否则任意 metric 字符串都会去扫索引（也挡住了把指标表当任意 KV 用）
var metricNames = map[string]bool{
	MetricSessionsActive: true, MetricSessionMaxElapsed: true, MetricLockWaitCount: true,
	MetricPingFail: true, MetricTLSCertDays: true, MetricSchemaDrift: true, MetricPitrReady: true,
}

// metricSamplesFromHealth 把一次健康快照转成指标点（纯函数：便于单测覆盖「不可观测就不写 0」）。
//
// 三条「不写」规则都是刻意的：
//   - 会话不可内省（方言无能力/库侧拒绝）：不写连接数与等锁数，否则图表显示 0 条连接；
//   - 没配证书：不写 tls_cert_days，0 会被读成「今天到期」；
//   - 没建过基线：不写漂移数，0 会被读成「结构很稳定」。
func metricSamplesFromHealth(instanceId uint64, health *dbi.DbHealth, at time.Time) []*entity.DbMetricSample {
	if health == nil {
		return nil
	}
	appendSample := func(list []*entity.DbMetricSample, metric string, value float64) []*entity.DbMetricSample {
		return append(list, &entity.DbMetricSample{InstanceId: instanceId, Metric: metric, Value: value, SampleTime: at})
	}

	samples := make([]*entity.DbMetricSample, 0, 6)
	pingFail := 0.0
	if health.PingErr != nil {
		pingFail = 1
	}
	samples = appendSample(samples, MetricPingFail, pingFail)

	if health.SessionsAvailable {
		samples = appendSample(samples, MetricSessionsActive, float64(len(health.Sessions)))
		var maxElapsedMs int64
		lockWaits := 0
		for _, session := range health.Sessions {
			if session == nil {
				continue
			}
			if session.ElapsedMs > maxElapsedMs {
				maxElapsedMs = session.ElapsedMs
			}
			if session.LockWait {
				lockWaits++
			}
		}
		samples = appendSample(samples, MetricSessionMaxElapsed, float64(maxElapsedMs)/1000)
		samples = appendSample(samples, MetricLockWaitCount, float64(lockWaits))
	}
	if health.TLSCertDaysLeft != nil {
		samples = appendSample(samples, MetricTLSCertDays, float64(*health.TLSCertDaysLeft))
	}
	if health.BaselineExists {
		samples = appendSample(samples, MetricSchemaDrift, float64(health.DriftedTables))
	}
	// PITR 前提：没探测到就不写点（写了 0 会被读成「已就绪」）
	if health.RecoveryPrepared != nil {
		pitrReady := 0.0
		if *health.RecoveryPrepared {
			pitrReady = 1
		}
		samples = appendSample(samples, MetricPitrReady, pitrReady)
	}
	return samples
}

// SampleAll 对所有实例采样一轮
func (a *dbMetricAppImpl) SampleAll(ctx context.Context) (int, error) {
	instances, err := a.instanceApp.ListByCond(model.NewCond())
	if err != nil {
		return 0, err
	}
	samples := make([]*entity.DbMetricSample, 0, len(instances)*4)
	now := time.Now()
	for _, instance := range instances {
		health, healthErr := a.instanceApp.Health(ctx, instance.Id)
		if healthErr != nil {
			// 单实例失败不中断整轮：一台库的凭证坏了不该让其他库没有曲线
			logx.WarnfContext(ctx, "sample db instance [%d] health failed: %s", instance.Id, healthErr.Error())
			continue
		}
		samples = append(samples, metricSamplesFromHealth(instance.Id, health, now)...)
	}
	if len(samples) == 0 {
		return 0, nil
	}
	// 写库用带系统身份的 ctx（审计列需要账号），但**取连接必须用不带登录态的 ctx**：
	// CanAccessByCode 的约定是「没有登录账号 = 系统内部任务，不做资源权限校验」，
	// 传一个 Id=0 的账号反而会被当成普通账号去校验并全部拒绝——采样于是全变 ping_fail
	opCtx := contextx.WithLoginAccount(ctx, &model.LoginAccount{Id: 0, Username: "system"})
	if err := a.Repo.BatchInsert(opCtx, samples); err != nil {
		return 0, err
	}
	return len(samples), nil
}

// Series 取趋势
func (a *dbMetricAppImpl) Series(ctx context.Context, instanceId uint64, metric string, minutes int) (*MetricSeries, error) {
	if !metricNames[metric] {
		return nil, errorx.NewBizI(ctx, imsg.ErrMetricUnsupported, "metric", metric)
	}
	if minutes <= 0 || minutes > int(metricRetention().Hours()*60) {
		minutes = 60
	}
	rows, err := a.Repo.Series(instanceId, metric, time.Now().Add(-time.Duration(minutes)*time.Minute))
	if err != nil {
		return nil, err
	}
	// 超过上限时按时间等距抽稀：保留首尾，图表趋势不变而传输量有界
	if len(rows) > metricSeriesMaxPoints {
		step := float64(len(rows)) / float64(metricSeriesMaxPoints)
		picked := make([]*entity.DbMetricSample, 0, metricSeriesMaxPoints)
		for i := 0; i < metricSeriesMaxPoints; i++ {
			picked = append(picked, rows[int(float64(i)*step)])
		}
		picked[len(picked)-1] = rows[len(rows)-1]
		rows = picked
	}
	points := make([]MetricPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, MetricPoint{Ts: row.SampleTime.UnixMilli(), Value: row.Value})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Ts < points[j].Ts })
	return &MetricSeries{Metric: metric, Points: points}, nil
}

// TimerSample 周期采样 + 过期清理
func (a *dbMetricAppImpl) TimerSample() {
	// 启动先采一轮：否则刚部署后 5 分钟内趋势页签永远是空的，看起来像功能坏了
	go func() {
		// 不带登录账号：系统内部任务的权限校验按约定直接放行
		if _, err := a.SampleAll(context.Background()); err != nil {
			logx.Warnf("initial db metric sampling failed: %s", err.Error())
		}
	}()

	// 调度固定 1 分钟一跳，是否真正采样由「距上次采样是否已过配置周期」决定：
	// 这样把周期从 5 分钟改成 1 分钟、或从 5 改成 30，都不需要重启进程才生效
	lastSampleAt := time.Now().Add(-metricSampleInterval())
	_ = taskx.BindCronTaskWithLock("db-metric-sample", "@every 1m", time.Minute, true, func() {
		if time.Since(lastSampleAt) < metricSampleInterval() {
			return
		}
		lastSampleAt = time.Now()
		ctx := context.Background()
		if _, err := a.SampleAll(ctx); err != nil {
			logx.ErrorfContext(ctx, "sample db metrics failed: %s", err.Error())
		}
		// 清理与采样同轮：过期数据的价值只为画图，留久了只会拖慢查询
		if rows, err := a.Repo.DeleteBefore(ctx, time.Now().Add(-metricRetention())); err != nil {
			logx.WarnfContext(ctx, "purge expired db metric samples failed: %s", err.Error())
		} else if rows > 0 {
			logx.Infof("purged %d expired db metric samples", rows)
		}
	})
}
