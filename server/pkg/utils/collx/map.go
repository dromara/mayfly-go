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
