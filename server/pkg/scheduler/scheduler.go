package scheduler

import (
	"errors"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

func init() {
	Start()
}

var (
	// SecondOptional 使秒字段可选，同时兼容 5 字段（标准 cron，如 "0 0 * * *"）
	// 与 6 字段（含秒，如 "0 0 3 * * ?"）两种表达式；保留 Descriptor 以支持 "@every ..." 等。
	// 解析器单独持有：注册与校验必须共用同一实例，否则会出现「校验放行、AddFun 报错」的语法漂移。
	specParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	cronService = cron.New(cron.WithParser(specParser))
	key2IdMap   collx.SM[string, cron.EntryID]
)

// ValidateSpec 校验 cron 表达式能否被本服务的解析器接受，不做注册。
//
// 由各任务的保存入口在落库前调用：表达式要到 AddFun 时才报错，而绑定失败仅写日志，
// 用户看到的是「保存成功、任务永不执行」。空表达式同样判错——被调度的任务没有表达式即永不触发。
// 返回的错误文本不含表达式本身，便于调用方嵌入自己的文案。
func ValidateSpec(spec string) error {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return errors.New("cron expression is empty")
	}
	if _, err := specParser.Parse(trimmed); err != nil {
		return err
	}
	return nil
}

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
