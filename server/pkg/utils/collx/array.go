package collx

// 数组比较
// 依次返回，新增值，删除值，以及不变值
func ArrayCompare[T comparable](newArr []T, oldArr []T) ([]T, []T, []T) {
	newSet := make(map[T]bool)
	oldSet := make(map[T]bool)

	// 将新数组和旧数组的元素分别添加到对应的哈希集合中
	for _, elem := range newArr {
		newSet[elem] = true
	}
	for _, elem := range oldArr {
		oldSet[elem] = true
	}

	var (
		added      []T
		deleted    []T
		unmodified []T
	)

	// 遍历新数组，根据元素是否存在于旧数组进行分类
	for _, elem := range newArr {
		if oldSet[elem] {
			unmodified = append(unmodified, elem)
		} else {
			added = append(added, elem)
		}
	}

	// 遍历旧数组，找出被删除的元素
	for _, elem := range oldArr {
		if !newSet[elem] {
			deleted = append(deleted, elem)
		}
	}

	return added, deleted, unmodified
}

// 判断数组中是否含有指定元素
func ArrayContains[T comparable](arr []T, el T) bool {
	for _, v := range arr {
		if v == el {
			return true
		}
	}
	return false
}

// 数组转为map
// @param keyFunc key的主键
func ArrayToMap[T any, K comparable](arr []T, keyFunc func(val T) K) map[K]T {
	res := make(map[K]T, len(arr))
	for _, val := range arr {
		key := keyFunc(val)
		res[key] = val
	}
	return res
}

// 数组映射，即将一数组元素通过映射函数转换为另一数组
func ArrayMap[T any, K comparable](arr []T, mapFunc func(val T) K) []K {
	res := make([]K, len(arr))
	for i, val := range arr {
		res[i] = mapFunc(val)
	}
	return res
}

// 将数组或切片按固定大小分割成小数组
func ArrayChunk[T any](arr []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(arr); i += chunkSize {
		end := i + chunkSize
		if end > len(arr) {
			end = len(arr)
		}
		chunks = append(chunks, arr[i:end])
	}
	return chunks
}

// 将数组切割为指定个数的子数组，并尽可能均匀
func ArraySplit[T any](arr []T, numGroups int) [][]T {
	if numGroups > len(arr) {
		numGroups = len(arr)
	}

	// 计算每个子数组的大小
	size := len(arr) / numGroups
	remainder := len(arr) % numGroups

	// 创建一个存放子数组的切片
	subArrays := make([][]T, numGroups)

	// 分割数组为子数组
	start := 0
	for i := range subArrays {
		subSize := size
		if i < remainder {
			subSize++
		}
		subArrays[i] = arr[start : start+subSize]
		start += subSize
	}

	return subArrays
}

// reduce操作
func ArrayReduce[T any, V any](arr []T, initialValue V, reducer func(V, T) V) V {
	value := initialValue
	for _, a := range arr {
		value = reducer(value, a)
	}
	return value
}

// 数组元素移除操作
func ArrayRemoveFunc[T any](arr []T, isDeleteFunc func(T) bool) []T {
	var newArr []T
	for _, a := range arr {
		if !isDeleteFunc(a) {
			newArr = append(newArr, a)
		}
	}
	return newArr
}
