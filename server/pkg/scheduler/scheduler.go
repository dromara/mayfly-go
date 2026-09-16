package scheduler

import (
	"errors"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/utils/collx"
	"time"

	"github.com/robfig/cron/v3"
)

func init() {
	Start()
}

var (
	cronService = cron.New(cron.WithSeconds())
	key2IdMap   collx.SM[string, cron.EntryID]
)

func Start() {
	cronService.Start()
}

func Stop() {
	cronService.Stop()
}

// Remove 根据任务id移除
func Remove(id cron.EntryID) {
	cronService.Remove(id)
}

// RemoveByKey 根据任务key移除
func RemoveByKey(key string) {
	logx.Debugf("remove cron func => [key = %s]", key)
	id, ok := key2IdMap.Load(key)
	if ok {
		Remove(id)
		key2IdMap.Delete(key)
	}
}

func GetCron() *cron.Cron {
	return cronService
}

// AddFun 添加任务
func AddFun(spec string, cmd func()) (cron.EntryID, error) {
	return cronService.AddFunc(spec, cmd)
}

// AddFunByKey 根据key添加定时任务
func AddFunByKey(key, spec string, cmd func()) error {
	logx.Debugf("add cron func => [key = %s]", key)
	if key == "" {
		return errors.New("scheduler key cannot be empty")
	}
	RemoveByKey(key)
	id, err := AddFun(spec, cmd)
	if err != nil {
		return err
	}
	key2IdMap.Store(key, id)
	return nil
}

// AddFunByKeyWithLock 添加带分布式锁的定时任务，支持多实例部署。
// 每次 cron 触发时尝试获取 Redis 分布式锁，获取成功才执行 cmd。
// 若 Redis 未配置（单机模式），则退化为普通执行。
// lockDuration 为锁的持有时间，应大于 cmd 最大执行时间。
func AddFunByKeyWithLock(key, spec string, lockDuration time.Duration, cmd func()) error {
	return AddFunByKey(key, spec, func() {
		lock := rediscli.NewLock("scheduler:"+key, lockDuration)
		if lock == nil {
			// Redis 未配置，单机模式直接执行
			cmd()
			return
		}
		if !lock.Lock() {
			logx.Debugf("[scheduler] skip cron job [%s], another instance is running", key)
			return
		}
		defer lock.UnLock()
		cmd()
	})
}

func ExistKey(key string) bool {
	_, ok := key2IdMap.Load(key)
	return ok
}
