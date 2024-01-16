package scheduler

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/logx"
	"sync"

	"github.com/robfig/cron/v3"
)

func init() {
	Start()
}

var (
	cronService = cron.New(cron.WithSeconds())
	key2IdMap   sync.Map
)

func Start() {
	cronService.Start()
}

func Stop() {
	cronService.Stop()
}

// 根据任务id移除
func Remove(id cron.EntryID) {
	cronService.Remove(id)
}

// 根据任务key移除
func RemoveByKey(key string) {
	logx.Debugf("移除cron任务 => [key = %s]", key)
	id, ok := key2IdMap.Load(key)
	if ok {
		Remove(id.(cron.EntryID))
		key2IdMap.Delete(key)
	}
}

func GetCron() *cron.Cron {
	return cronService
}

// 添加任务
func AddFun(spec string, cmd func()) cron.EntryID {
	id, err := cronService.AddFunc(spec, cmd)
	biz.ErrIsNilAppendErr(err, "添加任务失败: %s")
	return id
}

// 根据key添加定时任务
func AddFunByKey(key, spec string, cmd func()) {
	logx.Debugf("添加cron任务 => [key = %s]", key)
	RemoveByKey(key)
	key2IdMap.Store(key, AddFun(spec, cmd))
}

func ExistKey(key string) bool {
	_, ok := key2IdMap.Load(key)
	return ok
}
