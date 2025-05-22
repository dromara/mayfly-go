package pool

import (
	"errors"
	"time"
)

var ErrPoolClosed = errors.New("pool is closed")

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxConns            int           // 最大连接数
	IdleTimeout         time.Duration // 空闲连接超时时间
	WaitTimeout         time.Duration // 获取连接超时时间
	HealthCheckInterval time.Duration // 健康检查间隔
	OnPoolClose         func() error  // 连接池关闭时的回调
}

// Option 函数类型，用于配置 Pool
type Option func(*PoolConfig)

// WithMaxConns 设置最大连接数
func WithMaxConns(maxConns int) Option {
	return func(c *PoolConfig) {
		c.MaxConns = maxConns
	}
}

// WithIdleTimeout 设置空闲超时
func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *PoolConfig) {
		c.IdleTimeout = timeout
	}
}

// WithWaitTimeout 设置等待超时
func WithWaitTimeout(timeout time.Duration) Option {
	return func(c *PoolConfig) {
		c.WaitTimeout = timeout
	}
}

// WithHealthCheckInterval 设置健康检查间隔
func WithHealthCheckInterval(interval time.Duration) Option {
	return func(c *PoolConfig) {
		c.HealthCheckInterval = interval
	}
}

// WithOnPoolClose 设置连接池关闭回调
func WithOnPoolClose(fn func() error) Option {
	return func(c *PoolConfig) {
		c.OnPoolClose = fn
	}
}
