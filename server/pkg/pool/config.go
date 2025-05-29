package pool

import (
	"errors"
	"time"
)

var (
	ErrPoolClosed      = errors.New("pool is closed")
	ErrNoAvailableConn = errors.New("no available connection")
)

// PoolConfig 连接池配置
type PoolConfig[T Conn] struct {
	MaxConns            int           // 最大连接数
	IdleTimeout         time.Duration // 空闲连接超时时间
	WaitTimeout         time.Duration // 获取连接超时时间
	HealthCheckInterval time.Duration // 健康检查间隔
	OnPoolClose         func() error  // 连接池关闭时的回调

	OnConnClose func(conn T) error // 连接关闭时的回调,若err != nil则不关闭连接
}

// Option 函数类型，用于配置 Pool
type Option[T Conn] func(*PoolConfig[T])

// WithMaxConns 设置最大连接数
func WithMaxConns[T Conn](maxConns int) Option[T] {
	return func(c *PoolConfig[T]) {
		c.MaxConns = maxConns
	}
}

// WithIdleTimeout 设置空闲超时
func WithIdleTimeout[T Conn](timeout time.Duration) Option[T] {
	return func(c *PoolConfig[T]) {
		c.IdleTimeout = timeout
	}
}

// WithWaitTimeout 设置等待超时
func WithWaitTimeout[T Conn](timeout time.Duration) Option[T] {
	return func(c *PoolConfig[T]) {
		c.WaitTimeout = timeout
	}
}

// WithHealthCheckInterval 设置健康检查间隔
func WithHealthCheckInterval[T Conn](interval time.Duration) Option[T] {
	return func(c *PoolConfig[T]) {
		c.HealthCheckInterval = interval
	}
}

// WithOnPoolClose 设置连接池关闭回调
func WithOnPoolClose[T Conn](fn func() error) Option[T] {
	return func(c *PoolConfig[T]) {
		c.OnPoolClose = fn
	}
}

// WithOnConnClose 设置连接关闭回调, 若返回的错误不为nil，则不关闭连接
func WithOnConnClose[T Conn](fn func(conn T) error) Option[T] {
	return func(c *PoolConfig[T]) {
		c.OnConnClose = fn
	}
}

/**** GetOption Config ****/

// GetOption 用于配置 Get 的行为
type GetOption func(*getOptions)

// 控制 Get 行为的选项
type getOptions struct {
	updateLastActive bool // 是否更新 lastActive，默认 true
	newConn          bool // 连接不存在时是否创建新连接，默认 true
}

var (
	defaultGetOptions = getOptions{
		updateLastActive: true,
		newConn:          true,
	}
)

// WithGetNoUpdateLastActive 返回一个 Option，禁用更新 lastActive
func WithGetNoUpdateLastActive() GetOption {
	return func(o *getOptions) {
		o.updateLastActive = false
	}
}

// WithGetNoCreateConn 禁用获取时连接不存在创建连接
func WithGetNoNewConn() GetOption {
	return func(o *getOptions) {
		o.newConn = false
	}
}
