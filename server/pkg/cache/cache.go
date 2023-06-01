package cache

type Cache interface {
	// 添加缓存，如果缓存则返回错误
	Add(k string, v any) error

	// 如果不存在则添加缓存值，否则直接返回
	AddIfAbsent(k string, v any)

	// 如果存在则直接返回，否则调用getValue回调函数获取值并添加该缓存值
	// @return 缓存值
	ComputeIfAbsent(k string, getValueFunc func(string) (any, error)) (any, error)

	// 获取缓存值，参数1为值，参数2->是否存在该缓存
	Get(k string) (any, bool)

	// 缓存数量
	Count() int

	// 删除缓存
	Delete(k string)

	// 清空所有缓存
	Clear()
}
