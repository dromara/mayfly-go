package application

import (
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/alert/domain/entity"
)

// TestEvalTickNotCoarserThanMinInterval 调度粒度必须不粗于规则允许的最小评估间隔。
//
// shouldEvaluate 只能在调度点上判定「是否到期」，永远无法比调度更密。调度写死 @every 1m
// 而校验下限放到 10s，就会让 10~50s 的配置全部被静默拉长到 60s：界面写着「每隔 N 秒评估一次」、
// 提示语还劝用户「值越小告警越及时」，实际却一律 60 秒——配置项成了摆设
func TestEvalTickNotCoarserThanMinInterval(t *testing.T) {
	spec, ok := strings.CutPrefix(strings.TrimSpace(evalTickCron), "@every ")
	if !ok {
		t.Fatalf("evalTickCron=%q 不是 @every 形式，无法与最小评估间隔比对", evalTickCron)
	}
	tick, err := time.ParseDuration(spec)
	if err != nil {
		t.Fatalf("evalTickCron=%q 的时长不可解析: %v", evalTickCron, err)
	}
	if tick <= 0 {
		t.Fatalf("调度粒度必须为正: %v", tick)
	}
	if minEvalInterval < int(tick.Seconds()) {
		t.Fatalf("最小评估间隔 %ds 细于调度粒度 %v，该区间内的配置会被静默拉长到调度粒度", minEvalInterval, tick)
	}
	// 锁 TTL 必须不短于调度粒度，且覆盖单轮最坏耗时（基座契约：pkg/taskx/cron.go 与
	// pkg/scheduler/scheduler.go 都写「lockDuration 应大于 run 最大执行时间」）。
	//
	// 旧写法把这条判据写反了（断言 TTL < tick，理由是「否则每次触发会撞上自己上一轮的未过期锁」），
	// 而锁在 run 结束时由 defer 显式释放，根本不存在「撞自己」；按旧不变式，TTL=8s 碰上能跑 12s 的一轮，
	// 锁会在第 8 秒过期，另一实例在第 10 秒抢到同一把锁并发跑同一轮——lastEvalTime 与 pendingGroups
	// 都是进程内状态，于是同一告警评估两遍、通知发两遍
	if evalTickLockDuration < tick {
		t.Fatalf("锁 TTL %v 短于调度粒度 %v：一轮跑超 TTL 时另一实例可并发重跑同一轮，导致重复告警与重复通知", evalTickLockDuration, tick)
	}
	// flush 循环走同一个契约：它内部要真发 IM/邮件，不能拿更短的 TTL 特例化
	if groupFlushLockDuration < 10*time.Second {
		t.Fatalf("flush 锁 TTL %v 不足以覆盖一次带外发的刷新轮次", groupFlushLockDuration)
	}
}

// TestShouldEvaluateRespectsInterval 到期判定必须严格按规则的评估间隔放行，
// 且零值回落到默认间隔——它决定了细于调度粒度的配置不会退化成「每个调度点都评估」
func TestShouldEvaluateRespectsInterval(t *testing.T) {
	a := &alertEngineAppImpl{lastEvalTime: make(map[uint64]time.Time)}
	now := time.Now()

	rule := &entity.AlertRule{EvalInterval: 30}
	rule.Id = 1
	if !a.shouldEvaluate(rule, now) {
		t.Fatal("首次评估必须放行")
	}
	if a.shouldEvaluate(rule, now.Add(10*time.Second)) {
		t.Fatal("未到评估间隔不该重复评估")
	}
	if !a.shouldEvaluate(rule, now.Add(30*time.Second)) {
		t.Fatal("到达评估间隔必须再次放行")
	}

	zero := &entity.AlertRule{EvalInterval: 0}
	zero.Id = 2
	if !a.shouldEvaluate(zero, now) {
		t.Fatal("零间隔的首次评估必须放行")
	}
	if a.shouldEvaluate(zero, now.Add(59*time.Second)) {
		t.Fatalf("零间隔应回落到默认 %ds，却在 59s 就被放行", defaultEvalIntvl)
	}
	if !a.shouldEvaluate(zero, now.Add(time.Duration(defaultEvalIntvl)*time.Second)) {
		t.Fatalf("零间隔回落到默认 %ds 后应放行", defaultEvalIntvl)
	}
}

// TestEvaluateAllSkipsWhenPreviousRoundRunning 调度粒度可能细于单轮评估耗时，
// 而 cron 不会跳过仍在运行的任务：无自旋守卫会让多轮 evaluateAll 叠加、重复扫描规则表
func TestEvaluateAllSkipsWhenPreviousRoundRunning(t *testing.T) {
	a := &alertEngineAppImpl{}
	a.evalRunning.Store(true)
	// ruleApp 为 nil：一旦守卫失效进入主体就会空指针 panic，正好暴露「没跳过」
	a.evaluateAll()

	if !a.evalRunning.Load() {
		t.Fatal("跳过本轮时不得复位运行标记，否则等于放行了并发重入")
	}
}
