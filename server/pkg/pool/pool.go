package pool

import (
	"context"
)

// Conn 连接接口
// 连接池的连接必须实现 Conn 接口
type Conn interface {
	// Close 关闭连接
	Close() error

	// Ping 检查连接是否有效
	Ping() error
}

// Pool 连接池接口
type Pool[T Conn] interface {
	// 核心方法
	Get(ctx context.Context, opts ...GetOption) (T, error)
	Put(T) error
	Close()

	// 管理方法
	Resize(int)       // 动态调整大小
	Stats() PoolStats // 获取统计信息
}
