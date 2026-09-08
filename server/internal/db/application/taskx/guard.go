package taskx

import "sync"

// RunGuard 任务运行态原子守卫。
//
// 原实现基于 cache Set/Del 的 MarkRunning/IsRunning，IsRunning 检查与 MarkRunning
// 写入之间存在 check-then-act 竞态：两个触发源（手动执行与cron）同时通过检查后会
// 双重执行。本守卫基于 sync.Map.LoadOrStore 实现原子的"检查并标记"，仅第一个调用
// 方成功获取运行权。
//
// 单机部署下该标记为进程内状态，服务重启后自然清空（与 InitCronJob 重置运行态的
// 既有语义一致）；DB 中的 RunningState 字段仍负责面向 UI 的状态展示。
type RunGuard struct {
	running sync.Map // taskId -> struct{}
}

// Acquire 原子地标记任务为运行中。
// 返回 false 表示任务已在运行中（本次获取失败，调用方不应执行任务）。
func (g *RunGuard) Acquire(taskId uint64) bool {
	_, loaded := g.running.LoadOrStore(taskId, struct{}{})
	return !loaded
}

// Release 释放任务运行标记，无论任务当前是否标记为运行中均安全调用。
func (g *RunGuard) Release(taskId uint64) {
	g.running.Delete(taskId)
}

// IsRunning 判断任务是否处于运行中。
func (g *RunGuard) IsRunning(taskId uint64) bool {
	_, ok := g.running.Load(taskId)
	return ok
}
