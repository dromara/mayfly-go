package scheduler

import (
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

// Remove 根据任务id移除
func Remove(id cron.EntryID) {
	cronService.Remove(id)
}

// RemoveByKey 根据任务key移除
func RemoveByKey(key string) {
	logx.Debugf("remove cron func => [key = %s]", key)
	id, ok := key2IdMap.Load(key)
	if ok {
		Remove(id.(cron.EntryID))
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
	RemoveByKey(key)
	id, err := AddFun(spec, cmd)
	if err != nil {
		return err
	}
	key2IdMap.Store(key, id)
	return nil
}

func ExistKey(key string) bool {
	_, ok := key2IdMap.Load(key)
	return ok
}
