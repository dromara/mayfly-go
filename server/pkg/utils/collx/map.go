package collx

import (
	"database/sql/driver"
	"encoding/json"
	"mayfly-go/pkg/utils/anyx"

	"github.com/may-fly/cast"
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
