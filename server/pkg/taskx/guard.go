package taskx

import (
	"context"
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/utils/stringx"
	"sync"
	"time"
)

const (
	// runGuardKeyPrefix Redis 分布式锁 key 前缀
	runGuardKeyPrefix = "mayfly:run-guard:"

	// runGuardLockDuration 分布式锁 TTL。
	// 同步/迁移任务通常分钟级完成，2 小时 TTL 覆盖绝大多数场景。
	// 若任务执行时间超过 TTL，锁自动释放，另一实例可能接管——
	// 此时依赖 DB RunningState 做二次防护。
	runGuardLockDuration = 2 * time.Hour
)

// RunGuard 任务运行态原子守卫，支持多实例部署。
//
// 泛型参数 T 为任务标识类型（uint64 用于 DB 任务 ID，string 用于按 key 调度的任务等）。
//
// Redis 可用时通过 SETNX 实现跨实例的分布式互斥，确保同一任务在集群中
// 仅有一个实例执行；Redis 未配置时降级为进程内互斥（单机部署）。
//
// Release 通过 Lua 脚本原子校验所有权值后删除，防止误删其他实例的锁
// （与 RedisLock.UnLock 相同的安全模式）。
type RunGuard[T comparable] struct {
	// LockTTL 分布式锁 TTL，为零时使用默认值 runGuardLockDuration（2h）。
	// 短任务（如告警评估）可设较小值，长任务（如大数据同步）可设较大值。
	LockTTL time.Duration

	mu   sync.Mutex   // 保护 held map 的 check-and-mark 原子性
	held map[T]string // taskId → 所有权值（随机字符串）
}

// lockTTL 返回实际使用的锁 TTL。
// go-redis 将 sub-second duration 截断为 0 秒（Redis 中 0 = 永不过期），
// 因此强制最小 1 秒，防止亚秒级 TTL 导致锁永久泄漏。
func (g *RunGuard[T]) lockTTL() time.Duration {
	if g.LockTTL < time.Second {
		return runGuardLockDuration
	}
	return g.LockTTL
}

// Acquire 原子地标记任务为运行中。
// 返回 false 表示任务已在运行中（本次获取失败，调用方不应执行任务）。
func (g *RunGuard[T]) Acquire(taskId T) bool {
	_, ok := g.AcquireWithRunId(taskId)
	return ok
}

// AcquireWithRunId 与 Acquire 同为原子获取，但额外返回本次执行唯一标识（runId）。
// 日志、启动收尾、停止检查、水位推进等需围绕同一 runId 判定归属，避免“锁仍在，但已被新执行接管”
// 时旧日志被错杀、或旧批次在新实例已接管后继续写入。
func (g *RunGuard[T]) AcquireWithRunId(taskId T) (string, bool) {
	g.mu.Lock()
	if g.held == nil {
		g.held = make(map[T]string)
	}
	if _, exists := g.held[taskId]; exists {
		g.mu.Unlock()
		return "", false
	}

	val := stringx.Rand(32)

	if cli := rediscli.GetCli(); cli != nil {
		key := fmt.Sprintf("%s%v", runGuardKeyPrefix, taskId)
		ok, err := cli.SetNX(context.Background(), key, val, g.lockTTL()).Result()
		if err != nil {
			g.mu.Unlock()
			logx.Errorf("[RunGuard] redis SetNX failed for task %v: %v", taskId, err)
			return "", false
		}
		if !ok {
			g.mu.Unlock()
			return "", false
		}
		// SETNX 成功，持锁写入所有权值
		g.held[taskId] = val
		g.mu.Unlock()
		return val, true
	}

	// Redis 未配置：本地互斥即最终裁决
	g.held[taskId] = val
	g.mu.Unlock()
	return val, true
}

// Release 释放任务运行标记，无论任务当前是否标记为运行中均安全调用。
//
// Redis 可用时通过 CasDel（Lua 脚本）原子校验所有权值后删除，
// 防止误删其他实例的锁（如 TTL 过期后被其他实例获取的场景）。
func (g *RunGuard[T]) Release(taskId T) {
	g.mu.Lock()
	val, exists := g.held[taskId]
	if exists {
		delete(g.held, taskId)
	}
	g.mu.Unlock()

	if !exists {
		return
	}

	// Redis 可用时，原子校验所有权后删除
	if cli := rediscli.GetCli(); cli != nil {
		key := fmt.Sprintf("%s%v", runGuardKeyPrefix, taskId)
		rediscli.CasDel(key, val)
	}
}

// ReleaseWithRunId 仅当当前持有的 runId 与传入一致时才释放。
// 避免旧 runId 的收尾误删新持有者的锁（TTL 自然过期后新实例已接管的场景）。
func (g *RunGuard[T]) ReleaseWithRunId(taskId T, runId string) bool {
	g.mu.Lock()
	val, exists := g.held[taskId]
	if !exists || val != runId {
		g.mu.Unlock()
		return false
	}
	delete(g.held, taskId)
	g.mu.Unlock()

	if cli := rediscli.GetCli(); cli != nil {
		key := fmt.Sprintf("%s%v", runGuardKeyPrefix, taskId)
		rediscli.CasDel(key, runId)
	}
	return true
}

// CurrentRunId 返回当前持有任务锁的 runId（无持有则空串）。
// 优先本地 held（无延迟），本地无时回退 Redis GET（跨实例可见）。
func (g *RunGuard[T]) CurrentRunId(taskId T) string {
	g.mu.Lock()
	val, ok := g.held[taskId]
	g.mu.Unlock()
	if ok {
		return val
	}

	if cli := rediscli.GetCli(); cli != nil {
		key := fmt.Sprintf("%s%v", runGuardKeyPrefix, taskId)
		v, err := cli.Get(context.Background(), key).Result()
		if err != nil {
			// go-redis 对 key 不存在返回 redis.Nil，属正常无持有情形，不记错
			return ""
		}
		return v
	}

	return ""
}

// IsCurrentRun 判断传入的 runId 是否为任务当前持有者。
// 供批次循环、水位推进、收尾等写前确认“同一次执行”，避免被接管后旧任务继续写入。
func (g *RunGuard[T]) IsCurrentRun(taskId T, runId string) bool {
	if runId == "" {
		return false
	}
	return g.CurrentRunId(taskId) == runId
}

// IsRunning 判断任务是否处于运行中。
// 优先检查本地标记（零延迟），本地无标记时回退 Redis 检查（跨实例感知）。
func (g *RunGuard[T]) IsRunning(taskId T) bool {
	g.mu.Lock()
	_, ok := g.held[taskId]
	g.mu.Unlock()
	if ok {
		return true
	}

	// 本地无标记，检查 Redis（其他实例可能正在运行）
	if cli := rediscli.GetCli(); cli != nil {
		key := fmt.Sprintf("%s%v", runGuardKeyPrefix, taskId)
		exists, err := cli.Exists(context.Background(), key).Result()
		if err != nil {
			logx.Errorf("[RunGuard] redis exists failed for task %v: %v", taskId, err)
			return false
		}
		return exists > 0
	}

	return false
}
