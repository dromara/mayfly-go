package export

import (
	"fmt"
	"sort"
	"sync"
)

// ========== ConsumerRegistry：导出消费者注册中心 ==========

var (
	consumerRegistryMu sync.RWMutex
	consumerRegistry   = make(map[string]func() Consumer)
)

// Register 注册导出消费者（无状态，单例共享）。
// 适用于无可变状态的消费者（如 SQL、CSV）。
func Register(consumer Consumer) {
	if consumer == nil {
		panic("export: register nil consumer")
	}
	format := consumer.Format()
	consumerRegistryMu.Lock()
	defer consumerRegistryMu.Unlock()
	consumerRegistry[format] = func() Consumer {
		return consumer
	}
}

// RegisterFactory 注册导出消费者工厂函数。
// 适用于需要持有状态的消费者（如 JSON 需跟踪行计数），每次获取时创建新实例。
func RegisterFactory(format string, factory func() Consumer) {
	if factory == nil {
		panic("export: register nil consumer factory")
	}
	consumerRegistryMu.Lock()
	defer consumerRegistryMu.Unlock()
	consumerRegistry[format] = factory
}

// Get 获取指定格式的导出消费者。
// 对于通过 Register 注册的无状态消费者，返回同一实例；
// 对于通过 RegisterFactory 注册的有状态消费者，每次返回新实例。
func Get(format string) Consumer {
	consumerRegistryMu.RLock()
	defer consumerRegistryMu.RUnlock()
	factory := consumerRegistry[format]
	if factory == nil {
		return nil
	}
	return factory()
}

// Descriptor 导出格式描述符：注册中心对外的自描述信息，
// 上层（API/前端）据此动态拉取支持的格式清单，新增格式前端零改动
type Descriptor struct {
	Format        string // 格式标识（注册键，如 "sql"）
	Name          string // 人类可读名称（如 "SQL 脚本"）
	ContentType   string // MIME 类型（下载响应头）
	FileExtension string // 文件扩展名（下载文件名）
}

// Descriptors 返回全部已注册格式的描述符，按 Format 字典序排序（输出稳定）。
// 锁内仅快照注册键，factory() 在锁外调用——注册闭包属外部代码，持锁执行存在重入死锁风险
func Descriptors() []Descriptor {
	consumerRegistryMu.RLock()
	formats := make([]string, 0, len(consumerRegistry))
	for f := range consumerRegistry {
		formats = append(formats, f)
	}
	consumerRegistryMu.RUnlock()

	ds := make([]Descriptor, 0, len(formats))
	for _, format := range formats {
		c := Get(format)
		if c == nil {
			// 注册后运行期被移除的极端情况：跳过而非产出残缺描述符
			continue
		}
		ds = append(ds, Descriptor{
			Format:        format,
			Name:          c.Name(),
			ContentType:   c.ContentType(),
			FileExtension: c.FileExtension(),
		})
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].Format < ds[j].Format })
	return ds
}

// Formats 返回所有已注册的导出格式列表（按字典序排序）
func Formats() []string {
	consumerRegistryMu.RLock()
	defer consumerRegistryMu.RUnlock()
	formats := make([]string, 0, len(consumerRegistry))
	for f := range consumerRegistry {
		formats = append(formats, f)
	}
	sort.Strings(formats)
	return formats
}

// FormatError 不支持的导出格式错误
type FormatError struct {
	Format string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("unsupported export format: %s (registered formats: %v)", e.Format, Formats())
}
