// Package mask 提供数据库查询结果的字段脱敏能力（纯函数库，不依赖持久化）
package mask

import (
	"sync"

	"github.com/spf13/cast"
)

// Algorithm 脱敏算法接口，内置算法于 init 中真实注册，自定义算法可实现后调用 Register 注册
type Algorithm interface {
	// Name 算法唯一标识
	Name() string

	// Mask 对字符串值执行脱敏
	Mask(value string, params Params) string
}

// Params 算法参数，来源于规则/列标签的 params json 字段
type Params map[string]any

// Str 获取字符串参数，不存在时返回默认值
func (p Params) Str(key, def string) string {
	if v, ok := p[key]; ok {
		if s := cast.ToString(v); s != "" {
			return s
		}
	}
	return def
}

// Int 获取整数参数，不存在时返回默认值
func (p Params) Int(key string, def int) int {
	if v, ok := p[key]; ok {
		if i := cast.ToInt(v); i != 0 {
			return i
		}
	}
	return def
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Algorithm)
)

// Register 注册脱敏算法
func Register(a Algorithm) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[a.Name()] = a
}

// Get 获取指定名称的脱敏算法
func Get(name string) (Algorithm, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if a, ok := registry[name]; ok {
		return a, nil
	}
	return nil, ErrAlgorithmNotFound(name)
}

// Exists 判断算法是否已注册
func Exists(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := registry[name]
	return ok
}

// Names 返回所有已注册算法名
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
