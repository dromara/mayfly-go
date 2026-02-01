package collx

import (
	"database/sql/driver"
	"encoding/json"
	"mayfly-go/pkg/utils/anyx"
	"sync"

	"github.com/spf13/cast"
)

// M is a shortcut for map[string]any
type M map[string]any

// Set 设置key对应的值
func (m *M) Set(key string, val any) {
	if *m == nil {
		*m = M{}
	}
	(*m)[key] = val
}

// GetStr 获取key对应的string类型值
func (m M) GetStr(key string) string {
	return cast.ToString(m[key])
}

func (m M) GetStrSlice(key string) []string {
	return cast.ToStringSlice(m[key])
}

// GetInt 获取key对应的int类型值
func (m M) GetInt(key string) int {
	return cast.ToInt(m[key])
}

// GetInt64 获取key对应的int64类型值
func (m M) GetInt64(key string) int64 {
	return cast.ToInt64(m[key])
}

func (m M) GetFloat32(key string) float32 {
	return cast.ToFloat32(m[key])
}

func (m M) GetFloat64(key string) float64 {
	return cast.ToFloat64(m[key])
}

func (m M) GetBool(key string) bool {
	return cast.ToBool(m[key])
}

// Copy 复制一个新的map，在有线程并发读写时使用，避免并发读写导致数据不一致
func CopyM(m M) M {
	newMap := make(M)
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

/******************* thread-safe map *******************/

// SM 是一个线程安全的 map 实现，基于 sync.Map 并支持泛型
type SM[K comparable, V any] struct {
	m sync.Map
}

// NewSM 创建一个新的线程安全 map 实例
func NewSM[K comparable, V any]() *SM[K, V] {
	return &SM[K, V]{}
}

// Load 方法返回存储在 map 中键为 k 的值，如果不存在则返回零值和 false
func (sm *SM[K, V]) Load(k K) (V, bool) {
	v, ok := sm.m.Load(k)
	if !ok {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// Store 在 map 中存储键值对
func (sm *SM[K, V]) Store(k K, v V) {
	sm.m.Store(k, v)
}

// LoadOrStore 返回键 k 对应的值，如果不存在则存储指定的值
func (sm *SM[K, V]) LoadOrStore(k K, v V) (actual V, loaded bool) {
	actualVal, loaded := sm.m.LoadOrStore(k, v)
	return actualVal.(V), loaded
}

// LoadAndDelete 从 map 中删除键值对并返回之前的值
func (sm *SM[K, V]) LoadAndDelete(k K) (V, bool) {
	v, ok := sm.m.LoadAndDelete(k)
	if !ok {
		var zero V
		return zero, false
	}
	return v.(V), true
}

// Delete 从 map 中删除键值对
func (sm *SM[K, V]) Delete(k K) {
	sm.m.Delete(k)
}

// Clear 清空 map
func (sm *SM[K, V]) Clear() {
	sm.m.Clear()
}


// Range 遍历 map 中的所有键值对
func (sm *SM[K, V]) Range(f func(k K, v V) bool) {
	sm.m.Range(func(k, v any) bool {
		return f(k.(K), v.(V))
	})
}

// Len 返回 map 中键值对的数量
func (sm *SM[K, V]) Len() int {
	count := 0
	sm.Range(func(_ K, _ V) bool {
		count++
		return true
	})
	return count
}

// Values 返回 map 中所有的值
func (sm *SM[K, V]) Values() []V {
	values := make([]V, 0)
	sm.Range(func(_ K, v V) bool {
		values = append(values, v)
		return true
	})
	return values
}

/******************* M db driver *******************/
func (j *M) Scan(value any) error {
	if v, ok := value.([]byte); ok && len(v) > 0 {
		return json.Unmarshal(v, j)
	}
	return nil
}

func (m M) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

/******************* map utils *******************/

// 将偶数个元素转为对应的M (map[string]any)
//
// 偶数索引为key，奇数为value
func Kvs(elements ...any) M {
	myMap := make(map[string]any)

	for i := 0; i < len(elements); i += 2 {
		key := anyx.ToString(elements[i])
		if i+1 < len(elements) {
			value := elements[i+1]
			myMap[key] = value
		} else {
			myMap[key] = nil
		}
	}
	return myMap
}

// Keys returns the keys of the map m.
// The keys will be in an indeterminate order.
func MapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

// Values returns the values of the map m.
// The values will be in an indeterminate order.
func MapValues[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

// MapMerge maps merge, 若存在重复的key，则以最后的map值为准
func MapMerge[M ~map[K]V, K comparable, V any](maps ...M) M {
	mergedMap := make(M)

	for _, m := range maps {
		for k, v := range m {
			mergedMap[k] = v
		}
	}

	return mergedMap
}
