package cache

import (
	"fmt"
	global_cache "mayfly-go/pkg/cache"
	"strconv"
	"time"
)

const (
	firstBreachKey = "alert:first_breach:%d:%d"
	// 首次越界时间的兜底存活时长。该 TTL 仅用于防止规则删除后 key 泄漏：
	// 引擎在每个越界评估周期都会续写该值，因此正常运行的规则不会因过期而丢失首次越界时间。
	// TTL 必须大于规则评估间隔上限（alert_rule.go 的 maxEvalInterval = 86400s），
	// 否则长评估间隔的规则会在两次评估之间丢失首次越界时间并重新计时，导致永不触发。
	firstBreachTTL = 48 * time.Hour

	counterKeyPrefix = "alert:counter:%s"
	// 计数器存活时长：与首次越界时间同理，每次自增都会续期，仅用于规则删除/状态翻转后自动回收。
	// 必须大于评估间隔上限，否则 TriggerCount/RecoveryCount 会在计数途中被清零而永不达标
	counterTTL = 48 * time.Hour
)

// BreachTrackerImpl 首次越界时间缓存实现
type BreachTrackerImpl struct{}

func NewBreachTracker() *BreachTrackerImpl {
	return &BreachTrackerImpl{}
}

func (b *BreachTrackerImpl) GetFirstBreach(ruleId, resourceId uint64) time.Time {
	key := fmt.Sprintf(firstBreachKey, ruleId, resourceId)
	val := global_cache.GetStr(key)
	if val == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, val)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (b *BreachTrackerImpl) SetFirstBreach(ruleId, resourceId uint64, t time.Time) {
	key := fmt.Sprintf(firstBreachKey, ruleId, resourceId)
	_ = global_cache.Set(key, t.Format(time.RFC3339Nano), firstBreachTTL)
}

func (b *BreachTrackerImpl) DelFirstBreach(ruleId, resourceId uint64) {
	key := fmt.Sprintf(firstBreachKey, ruleId, resourceId)
	global_cache.Del(key)
}

// ---- ConsecutiveCounter 实现（通过 cache.Incr 原子操作，支持多实例） ----

type ConsecutiveCounterImpl struct{}

func NewConsecutiveCounter() *ConsecutiveCounterImpl {
	return &ConsecutiveCounterImpl{}
}

func (c *ConsecutiveCounterImpl) Get(key string) int64 {
	cacheKey := fmt.Sprintf(counterKeyPrefix, key)
	val := global_cache.GetStr(cacheKey)
	if val == "" {
		return 0
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// Incr 通过 IncrWithTTL 原子自增并续期，Redis 模式下天然支持多实例并发安全，
// LocalCache 模式下通过进程内 mutex 保证原子性。
// 必须使用带 TTL 的自增：裸 INCR 创建的 key 永不过期，规则删除或进程重启后计数器会永久残留
func (c *ConsecutiveCounterImpl) Incr(key string) int64 {
	cacheKey := fmt.Sprintf(counterKeyPrefix, key)
	n, err := global_cache.IncrWithTTL(cacheKey, counterTTL)
	if err != nil {
		return 0
	}
	return n
}

func (c *ConsecutiveCounterImpl) Reset(key string) {
	cacheKey := fmt.Sprintf(counterKeyPrefix, key)
	global_cache.Del(cacheKey)
}
