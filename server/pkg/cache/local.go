package cache

import (
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
)

type LocalCache struct {
	defaultCache

	tm     *TimedCache
	incrMu sync.Mutex // 保护 Incr 的 read-modify-write 原子性
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

// Incr 原子自增缓存中的整数值（本地实现，通过 mutex 保证原子性）
// 若 key 不存在则初始化为 0 后自增（返回 1），使用默认过期时间。
func (lc *LocalCache) Incr(key string) (int64, error) {
	lc.incrMu.Lock()
	defer lc.incrMu.Unlock()

	// 尝试通过 TimedCache 的 Increment 原子自增（保留原有过期时间）
	if err := lc.tm.Increment(key, 1); err == nil {
		if val, ok := lc.tm.Get(key); ok {
			return cast.ToInt64(val), nil
		}
	}

	// key 不存在或类型不匹配，初始化为 1
	_ = lc.tm.Add(key, int64(1), lc.tm.defaultExpiration)
	return 1, nil
}

// IncrWithTTL 自增并刷新过期时间。
// 与 Incr 的区别：TimedCache.Increment 会保留 key 原有的过期时间，长期自增的计数器
// 会在首次写入的过期时间到达时被清除，因此这里在读改写锁内直接以新 TTL 覆写。
func (lc *LocalCache) IncrWithTTL(key string, ttl time.Duration) (int64, error) {
	lc.incrMu.Lock()
	defer lc.incrMu.Unlock()

	var val int64
	if got, ok := lc.tm.Get(key); ok {
		val = cast.ToInt64(got)
	}
	val++
	if err := lc.tm.Add(key, val, ttl); err != nil {
		return 0, err
	}
	return val, nil
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
