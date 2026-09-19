package taskx

import (
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/scheduler"
	"time"
)

// BindCronTask 按 key 重新绑定定时任务：先移除旧绑定，enabled 为 true 时注册新任务。
//
// 返回注册错误（key 为空、cron 表达式非法等）；enabled 为 false 时仅做移除，恒返回 nil。
func BindCronTask(key, spec string, enabled bool, run func()) error {
	scheduler.RemoveByKey(key)
	if !enabled {
		return nil
	}
	if err := scheduler.AddFunByKey(key, spec, run); err != nil {
		logx.ErrorTrace("bind cron task failed", err)
		return err
	}
	return nil
}

// BindCronTaskWithLock 绑定带分布式锁的定时任务，支持多实例部署。
//
// 每次 cron 触发时尝试获取 Redis 分布式锁，获取成功才执行 run；
// 若 Redis 未配置（单机模式），则退化为普通执行。
// lockDuration 为锁的持有时间，应大于 run 最大执行时间。
func BindCronTaskWithLock(key, spec string, lockDuration time.Duration, enabled bool, run func()) error {
	scheduler.RemoveByKey(key)
	if !enabled {
		return nil
	}
	if err := scheduler.AddFunByKeyWithLock(key, spec, lockDuration, run); err != nil {
		logx.ErrorTrace("bind cron task with lock failed", err)
		return err
	}
	return nil
}

// UnbindCronTask 解绑定时任务（任务删除或停用时调用）。
func UnbindCronTask(key string) {
	scheduler.RemoveByKey(key)
}
