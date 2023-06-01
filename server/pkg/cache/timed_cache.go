package cache

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Item struct {
	Value      any   // 对象
	Expiration int64 // 缓存有效时间
	UseCount   int64 // 使用次数
	AccessTime int64 // 访问时间
}

// 是否过期
func (item Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.AccessTime+item.Expiration
}

// 是否过期
// @return 值 and 是否过期
func (item *Item) GetValue(updateAccessTime bool) (any, bool) {
	isExpired := item.Expired()
	// 更新最后访问时间，用于增加值的有效期
	if !isExpired && updateAccessTime {
		item.AccessTime = time.Now().UnixNano()
	}
	item.UseCount = item.UseCount + 1
	return item.Value, isExpired
}

const (
	// 无过期时间
	NoExpiration time.Duration = -1
	// 默认过期时间
	DefaultExpiration time.Duration = 0
	// 默认清理缓存时间差
	DefaultCleanupInterval = 10
)

type TimedCache struct {
	*timedcache
}

type timedcache struct {
	defaultExpiration time.Duration
	updateAccessTime  bool // 是否更新最后访问时间
	items             map[any]*Item
	mu                sync.RWMutex
	onEvicted         func(any, any) // 移除时回调函数
	janitor           *janitor
}

// Add an item to the cache only if an item doesn't already exist for the given
// key, or if the existing item has expired. Returns an error otherwise.
func (c *timedcache) Add(k any, x any, d time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, found := c.get(k)
	if found {
		return fmt.Errorf("Item %s already exists", k)
	}
	c.set(k, x, d)
	return nil
}

func (c *timedcache) Put(k any, x any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.set(k, x, c.defaultExpiration)
}

func (c *timedcache) AddIfAbsent(k any, x any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, found := c.get(k)
	if found {
		return
	}
	c.set(k, x, c.defaultExpiration)
}

func (c *timedcache) ComputeIfAbsent(k any, getValueFunc func(any) (any, error)) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, found := c.get(k)
	if found {
		return value, nil
	}
	value, err := getValueFunc(k)
	if err != nil {
		return nil, err
	}

	c.set(k, value, c.defaultExpiration)
	return value, nil
}

func (c *timedcache) set(k any, x any, d time.Duration) {
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = d.Nanoseconds()
	}
	c.items[k] = &Item{
		Value:      x,
		Expiration: e,
		AccessTime: time.Now().UnixNano(),
	}
}

// Get an item from the cache. Returns the item or nil, and a bool indicating
// whether the key was found.
func (c *timedcache) Get(k any) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.get(k)
}

func (c *timedcache) get(k any) (any, bool) {
	item, found := c.items[k]
	if !found {
		return nil, false
	}

	value, expired := item.GetValue(c.updateAccessTime)
	if expired {
		// c.Delete(k)
		return nil, false
	}
	return value, true
}

// Increment an item of type int, int8, int16, int32, int64, uintptr, uint,
// uint8, uint32, or uint64, float32 or float64 by n. Returns an error if the
// item's value is not an integer, if it was not found, or if it is not
// possible to increment it by n. To retrieve the incremented value, use one
// of the specialized methods, e.g. IncrementInt64.
func (c *timedcache) Increment(k any, n int64) error {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return fmt.Errorf("Item %s not found", k)
	}
	switch v.Value.(type) {
	case int:
		v.Value = v.Value.(int) + int(n)
	case int8:
		v.Value = v.Value.(int8) + int8(n)
	case int16:
		v.Value = v.Value.(int16) + int16(n)
	case int32:
		v.Value = v.Value.(int32) + int32(n)
	case int64:
		v.Value = v.Value.(int64) + n
	case uint:
		v.Value = v.Value.(uint) + uint(n)
	case uintptr:
		v.Value = v.Value.(uintptr) + uintptr(n)
	case uint8:
		v.Value = v.Value.(uint8) + uint8(n)
	case uint16:
		v.Value = v.Value.(uint16) + uint16(n)
	case uint32:
		v.Value = v.Value.(uint32) + uint32(n)
	case uint64:
		v.Value = v.Value.(uint64) + uint64(n)
	case float32:
		v.Value = v.Value.(float32) + float32(n)
	case float64:
		v.Value = v.Value.(float64) + float64(n)
	default:
		c.mu.Unlock()
		return fmt.Errorf("The value for %s is not an integer", k)
	}
	c.items[k] = v
	c.mu.Unlock()
	return nil
}

// Returns the number of items in the cache. This may include items that have
// expired, but have not yet been cleaned up.
func (c *timedcache) Count() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// Copies all unexpired items in the cache into a new map and returns it.
func (c *timedcache) Items() map[any]*Item {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[any]*Item, len(c.items))
	now := time.Now().UnixNano()
	for k, v := range c.items {
		// "Inlining" of Expired
		if v.Expiration > 0 {
			if now > (v.Expiration + v.AccessTime) {
				continue
			}
		}
		m[k] = v
	}
	return m
}

// 删除指定key的数据
func (c *timedcache) Delete(k any) {
	c.mu.Lock()
	v, evicted := c.delete(k)
	c.mu.Unlock()
	if evicted {
		c.onEvicted(k, v)
	}
}

func (c *timedcache) delete(k any) (any, bool) {
	// 如果有移除回调函数，则返回值及是否有删除回调函数用于进行回调处理
	if c.onEvicted != nil {
		if v, found := c.items[k]; found {
			delete(c.items, k)
			return v.Value, true
		}
	}
	delete(c.items, k)
	return nil, false
}

type keyAndValue struct {
	key   any
	value any
}

// Delete all expired items from the cache.
func (c *timedcache) DeleteExpired() {
	var evictedItems []keyAndValue
	now := time.Now().UnixNano()
	c.mu.Lock()
	for k, v := range c.items {
		// "Inlining" of expired
		if v.Expiration > 0 && now > (v.Expiration+v.AccessTime) {
			ov, evicted := c.delete(k)
			if evicted {
				evictedItems = append(evictedItems, keyAndValue{k, ov})
			}
		}
	}
	c.mu.Unlock()
	for _, v := range evictedItems {
		c.onEvicted(v.key, v.value)
	}
}

// 清空所有缓存
func (c *timedcache) Clear() {
	c.mu.Lock()
	c.items = map[any]*Item{}
	c.mu.Unlock()
}

// Write the cache's items (using Gob) to an io.Writer.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *timedcache) Save(w io.Writer) (err error) {
	enc := gob.NewEncoder(w)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Error registering item types with Gob library")
		}
	}()
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, v := range c.items {
		gob.Register(v.Value)
	}
	err = enc.Encode(&c.items)
	return
}

// Save the cache's items to the given filename, creating the file if it
// doesn't exist, and overwriting it if it does.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *timedcache) SaveFile(fname string) error {
	fp, err := os.Create(fname)
	if err != nil {
		return err
	}
	err = c.Save(fp)
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

// Add (Gob-serialized) cache items from an io.Reader, excluding any items with
// keys that already exist (and haven't expired) in the current cache.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *timedcache) Load(r io.Reader) error {
	dec := gob.NewDecoder(r)
	items := map[string]*Item{}
	err := dec.Decode(&items)
	if err == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		for k, v := range items {
			ov, found := c.items[k]
			if !found || ov.Expired() {
				c.items[k] = v
			}
		}
	}
	return err
}

// Load and add cache items from the given filename, excluding any items with
// keys that already exist in the current cache.
//
// NOTE: This method is deprecated in favor of c.Items() and NewFrom() (see the
// documentation for NewFrom().)
func (c *timedcache) LoadFile(fname string) error {
	fp, err := os.Open(fname)
	if err != nil {
		return err
	}
	err = c.Load(fp)
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *timedcache) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *TimedCache) {
	c.janitor.stop <- true
}

func runJanitor(c *timedcache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}

func newCache(de time.Duration, m map[any]*Item) *timedcache {
	if de == 0 {
		de = -1
	}
	c := &timedcache{
		defaultExpiration: de,
		items:             m,
	}
	return c
}

func newCacheWithJanitor(de time.Duration, ci time.Duration, m map[any]*Item) *TimedCache {
	c := newCache(de, m)
	// This trick ensures that the janitor goroutine (which--granted it
	// was enabled--is running DeleteExpired on c forever) does not keep
	// the returned C object from being garbage collected. When it is
	// garbage collected, the finalizer stops the janitor goroutine, after
	// which c can be collected.
	C := &TimedCache{c}
	if ci > 0 {
		runJanitor(c, ci)
		// runtime.SetFinalizer(C, stopJanitor)
	}
	return C
}

// Return a new cache with a given default expiration duration and cleanup
// interval. If the expiration duration is less than one (or NoExpiration),
// the items in the cache never expire (by default), and must be deleted
// manually. If the cleanup interval is less than one, expired items are not
// deleted from the cache before calling c.DeleteExpired().
func NewTimedCache(defaultExpiration, cleanupInterval time.Duration) *TimedCache {
	items := make(map[any]*Item)
	return newCacheWithJanitor(defaultExpiration, cleanupInterval, items)
}

// 调用删除函数时，会回调该剔除函数
func (c *TimedCache) OnEvicted(f func(any, any)) *TimedCache {
	c.mu.Lock()
	c.onEvicted = f
	c.mu.Unlock()
	return c
}

// 是否更新最后访问时间，是则会更新最后访问时间
// 即只要在指定缓存时间内都没有访问该缓存，则会失效，反之失效开始时间点为最后访问时间
func (c *TimedCache) WithUpdateAccessTime(update bool) *TimedCache {
	c.mu.Lock()
	c.updateAccessTime = update
	c.mu.Unlock()
	return c
}
