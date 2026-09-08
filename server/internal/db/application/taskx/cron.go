package taskx

import (
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/scheduler"
)

// BindCronTask 按 key 重新绑定定时任务：先移除旧绑定，enabled 为 true 时注册新任务。
//
// 统一迁移/同步任务原先各自复制的 addCronJob 逻辑（移除旧任务 + 按启用状态重注册）。
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

// UnbindCronTask 解绑定时任务（任务删除或停用时调用）。
func UnbindCronTask(key string) {
	scheduler.RemoveByKey(key)
}
