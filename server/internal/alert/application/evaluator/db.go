package evaluator

import (
	"context"
	"fmt"
	"sync"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/internal/alert/imsg"
	dbapp "mayfly-go/internal/db/application"
	"mayfly-go/internal/db/dbm/dbi"
	pkgconsts "mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

// 数据库实例指标名
const (
	// MetricDbPingFail 实例不可达（1 不可达 / 0 可达）：存活类指标，取不到样本也能给出结论
	MetricDbPingFail = "db_ping_fail"
	// MetricDbSessionsActive 活动会话数
	MetricDbSessionsActive = "db_sessions_active"
	// MetricDbSessionMaxElapsed 最长会话已执行秒数（「跑太久的查询」用阈值表达，平台不内置慢查询门限）
	MetricDbSessionMaxElapsed = "db_session_max_elapsed"
	// MetricDbLockWaitCount 处于等锁状态的会话数
	MetricDbLockWaitCount = "db_lock_wait_count"
	// MetricDbSchemaDrift 与结构基线不一致的表数（不依赖会话内省能力，来自主自己的基线表）
	MetricDbSchemaDrift = "db_schema_drift_tables"
)

// DbEvaluator 数据库实例指标评估器。
//
// 数据源不新建采集底座：可达性用连接池现成的 Ping，会话事实来自 db 模块的 SessionInspector
// （见 dbi.DbHealth 注释），因此 DB 指标与真库能力严格一致——方言没实现的能力不谎报为 0
type DbEvaluator struct {
	dbInstanceApp dbapp.Instance `inject:"T"`

	// metrics 指标注册表：key=指标名，value=提取器
	metrics metricRegistry[*dbi.DbHealth]
	// metricsOnce 注册表只初始化一次：评估器为单例，引擎并发调用 Metrics/Evaluate
	metricsOnce sync.Once
}

var _ service.AlertEvaluator = (*DbEvaluator)(nil)

func init() {
	registerEvaluator(&DbEvaluator{})
}

// ensureMetrics 懒初始化指标注册表
func (e *DbEvaluator) ensureMetrics() {
	e.metricsOnce.Do(e.initMetrics)
}

func (e *DbEvaluator) initMetrics() {
	e.metrics = make(metricRegistry[*dbi.DbHealth])

	// 不可达：health 采样本身失败时也按不可达处理（实例被删除/连接配置错误都属同一后果）
	e.metrics.register(&MetricExtractor[*dbi.DbHealth]{
		Definition: service.MetricDefinition{Key: MetricDbPingFail, Label: "alert.metricDbPingFail", LabelMsgId: imsg.MetricDbPingFail, Unit: "", HasRange: true, Min: 0, Max: 1},
		Extract:    extractDbPingFail,
	})

	e.metrics.register(&MetricExtractor[*dbi.DbHealth]{
		Definition: service.MetricDefinition{Key: MetricDbSessionsActive, Label: "alert.metricDbSessionsActive", LabelMsgId: imsg.MetricDbSessionsActive, Unit: "", HasRange: false, Min: 0},
		Extract:    extractDbSessionsActive,
	})

	e.metrics.register(&MetricExtractor[*dbi.DbHealth]{
		Definition: service.MetricDefinition{Key: MetricDbSessionMaxElapsed, Label: "alert.metricDbSessionMaxElapsed", LabelMsgId: imsg.MetricDbSessionMaxElapsed, Unit: "s", HasRange: false, Min: 0},
		Extract:    extractDbSessionMaxElapsed,
	})

	e.metrics.register(&MetricExtractor[*dbi.DbHealth]{
		Definition: service.MetricDefinition{Key: MetricDbLockWaitCount, Label: "alert.metricDbLockWaitCount", LabelMsgId: imsg.MetricDbLockWaitCount, Unit: "", HasRange: false, Min: 0},
		Extract:    extractDbLockWaitCount,
	})

	e.metrics.register(&MetricExtractor[*dbi.DbHealth]{
		Definition: service.MetricDefinition{Key: MetricDbSchemaDrift, Label: "alert.metricDbSchemaDrift", LabelMsgId: imsg.MetricDbSchemaDrift, Unit: "", HasRange: false, Min: 0},
		Extract:    extractDbSchemaDrift,
	})
}

// extractDbSchemaDrift 漂移表数。
//
// 没建过基线时**返回错误而不是 0**：0 会被规则当成「结构很稳定」并在界面上标绿，
// 而事实是根本没得比（与会话类指标区分能力缺失同一取向）
func extractDbSchemaDrift(health *dbi.DbHealth, healthErr error) (float64, error) {
	if healthErr != nil {
		return 0, fmt.Errorf("db health unavailable: %w", healthErr)
	}
	if health == nil || !health.BaselineExists {
		return 0, fmt.Errorf("no table structure baseline has been taken for this db instance")
	}
	return float64(health.DriftedTables), nil
}

// extractDbPingFail 可达性指标：取不到样本(healthErr)与 Ping 失败同样是「不可达」，返回 1 而非错误；
// 可达返回 0。与机器 status 指标同一取向：存活类指标在数据源失败时仍要有结论
func extractDbPingFail(health *dbi.DbHealth, healthErr error) (float64, error) {
	if healthErr != nil || health == nil || health.PingErr != nil {
		return 1, nil
	}
	return 0, nil
}

// requireSessions 会话类指标的共同前置：库不可达或方言不支持会话内省时**返回错误**而非 0。
//
// 返回 0 会让「观测不到」在界面上看起来像「一切正常」，并让规则可能被误判恢复
func requireSessions(health *dbi.DbHealth, healthErr error) error {
	if healthErr != nil {
		return fmt.Errorf("db health unavailable: %w", healthErr)
	}
	if health == nil {
		return fmt.Errorf("db health is empty")
	}
	if health.PingErr != nil {
		return fmt.Errorf("db instance unreachable: %w", health.PingErr)
	}
	if !health.SessionsAvailable {
		return fmt.Errorf("db instance does not support session introspection")
	}
	return nil
}

func extractDbSessionsActive(health *dbi.DbHealth, healthErr error) (float64, error) {
	if err := requireSessions(health, healthErr); err != nil {
		return 0, err
	}
	return float64(len(health.Sessions)), nil
}

// extractDbSessionMaxElapsed 最长会话已执行秒数。
//
// 不叫「慢查询数」是因为门限该由规则表达（不同环境差异大），平台只给事实：
// 阈值 30 秒与 300 秒的规则可以共用同一个指标
func extractDbSessionMaxElapsed(health *dbi.DbHealth, healthErr error) (float64, error) {
	if err := requireSessions(health, healthErr); err != nil {
		return 0, err
	}
	var maxMs int64
	for _, s := range health.Sessions {
		if s.ElapsedMs > maxMs {
			maxMs = s.ElapsedMs
		}
	}
	return float64(maxMs) / 1000, nil
}

// extractDbLockWaitCount 等锁会话数（判据由各方言在 ListSessions 内给出，见 dbi.SessionInfo）
func extractDbLockWaitCount(health *dbi.DbHealth, healthErr error) (float64, error) {
	if err := requireSessions(health, healthErr); err != nil {
		return 0, err
	}
	var count int
	for _, s := range health.Sessions {
		if s.LockWait {
			count++
		}
	}
	return float64(count), nil
}

func (e *DbEvaluator) ResourceType() int8 {
	return pkgconsts.ResourceTypeDbInstance
}

func (e *DbEvaluator) Metrics() []service.MetricDefinition {
	e.ensureMetrics()
	return e.metrics.definitions()
}

func (e *DbEvaluator) ResourceExists(resourceId uint64) bool {
	instance, err := e.dbInstanceApp.GetById(resourceId)
	return err == nil && instance != nil
}

func (e *DbEvaluator) ResourceName(resourceId uint64) string {
	instance, err := e.dbInstanceApp.GetById(resourceId)
	if err != nil || instance == nil {
		logx.Warnf("[alert] get db instance[%d] name error: %v", resourceId, err)
		return ""
	}
	return instance.Name
}

func (e *DbEvaluator) ResourceIdsByCodes(codes []string) []uint64 {
	if len(codes) == 0 {
		return nil
	}
	instances, err := e.dbInstanceApp.ListByCond(model.NewCond().In("code", codes), "id")
	if err != nil {
		logx.Errorf("[alert] list db instances by codes error: %s", err.Error())
		return nil
	}
	ids := make([]uint64, 0, len(instances))
	for _, in := range instances {
		ids = append(ids, in.Id)
	}
	return ids
}

// Evaluate 评估数据库实例告警条件；下标对齐与可靠性语义与机器评估器一致（见 contract checklist）
func (e *DbEvaluator) Evaluate(ctx context.Context, param *service.EvalParam) (*service.EvalResult, error) {
	e.ensureMetrics()

	condition := param.Rule.Condition
	if condition == nil || len(condition.Items) == 0 {
		return &service.EvalResult{Evaluated: true}, nil
	}

	health, healthErr := e.dbInstanceApp.Health(ctx, param.ResourceId)
	// 采样失败：只有「可达性」这一存活类指标还能给出结论，其余条件一律不可判定
	if healthErr != nil && !usesMetric(condition, MetricDbPingFail) {
		logx.Warnf("[alert] db instance[%d] health unavailable: %s, evaluation unreliable", param.ResourceId, healthErr.Error())
		return &service.EvalResult{Evaluated: false}, nil
	}

	result := &service.EvalResult{Evaluated: true}
	for _, item := range condition.Items {
		eval := service.ConditionEval{
			Metric:    item.Metric,
			Threshold: item.Value,
			Compare:   item.Compare,
		}

		currentValue, err := e.metrics.extract(health, healthErr, item.Metric)
		switch {
		case err != nil:
			// 下标保持对齐：记录失败原因而非跳过该条件
			eval.Err = err.Error()
			logx.Warnf("[alert] db instance[%d] metric[%s] eval error: %s", param.ResourceId, item.Metric, err.Error())
		case !entity.IsValidCompare(item.Compare):
			eval.Err = fmt.Sprintf("invalid compare operator: %s", item.Compare)
			logx.Warnf("[alert] db instance[%d] metric[%s] invalid compare operator: %s", param.ResourceId, item.Metric, item.Compare)
		default:
			eval.CurrentValue = currentValue
			eval.Satisfied = compareValue(currentValue, item.Compare, item.Value)
		}
		result.Items = append(result.Items, eval)
	}

	result.Triggered = isTriggered(condition.Operator, result.Items)
	result.Evaluated = result.Triggered || !hasUnevaluableItem(result.Items)
	return result, nil
}
