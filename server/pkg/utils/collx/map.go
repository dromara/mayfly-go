package collx

import "mayfly-go/pkg/utils/anyx"

// M is a shortcut for map[string]any
type M map[string]any

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
