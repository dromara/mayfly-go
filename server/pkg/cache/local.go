package cache

import (
	"strings"
	"time"

	"github.com/may-fly/cast"
)

type LocalCache struct {
	defaultCache

	tm *TimedCache
}

var _ (Cache) = (*LocalCache)(nil)

func NewLocalCache() *LocalCache {
	lc := &LocalCache{
		tm: NewTimedCache(time.Minute*time.Duration(5), 30*time.Second),
	}
	lc.c = lc
	return lc
}

func (lc *LocalCache) Set(key string, value any, duration time.Duration) error {
	return lc.tm.Add(key, value, duration)
}

func (lc *LocalCache) Get(k string) (any, bool) {
	return lc.tm.Get(k)
}

func (lc *LocalCache) Delete(k string) error {
	lc.tm.Delete(k)
	return nil
}

func (lc *LocalCache) DeleteByKeyPrefix(keyPrefix string) error {
	for key := range lc.tm.Items() {
		if strings.HasPrefix(cast.ToString(key), keyPrefix) {
			lc.tm.Delete(key)
		}
	}
	return nil
}

func (lc *LocalCache) Count() int {
	return lc.tm.Count()
}

func (lc *LocalCache) Clear() {
	lc.tm.Clear()
}
