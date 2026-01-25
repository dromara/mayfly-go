package starter

import (
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Option 是用于配置启动选项的函数类型
type Option func(*Options)

// Options 包含所有启动时的配置选项
type Options struct {
	// 日志保存器
	LogSaver func() req.SaveLogFunc

	// 数据库初始化相关回调
	OnDbReady func(db *gorm.DB) error

	// 路由注册完成回调
	OnRoutesReady func(engine *gin.Engine)

	// 服务启动相关回调
	OnBeforeStart func()

	// 静态资源路由配置
	StaticRouter *StaticRouter
}

// WithLogSaver 设置日志保存器
func WithLogSaver(saver func() req.SaveLogFunc) Option {
	return func(o *Options) {
		o.LogSaver = saver
	}
}

// WithOnDbReady 设置数据库准备就绪回调函数
func WithOnDbReady(fn func(db *gorm.DB) error) Option {
	return func(o *Options) {
		o.OnDbReady = fn
	}
}

// WithOnRoutesReady 设置路由准备就绪回调函数
func WithOnRoutesReady(fn func(engine *gin.Engine)) Option {
	return func(o *Options) {
		o.OnRoutesReady = fn
	}
}

// WithOnBeforeStart 设置服务启动前回调函数
func WithOnBeforeStart(fn func()) Option {
	return func(o *Options) {
		o.OnBeforeStart = fn
	}
}

// WithStaticRouter 添加静态资源路由
func WithStaticRouter(staticRouter *StaticRouter) Option {
	return func(o *Options) {
		o.StaticRouter = staticRouter
	}
}

// NewOptions 创建默认的选项配置
func NewOptions(opts ...Option) *Options {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}
