package session

// GetOptions 获取或创建会话的配置选项
type GetOptions struct {
	messageLimit    int  // 消息条数限制，0 表示加载全部
	skipWindowCheck bool // 跳过窗口检查（宿主已有含 preamble 口径的完整裁剪时避免双层触发）
}

// GetOptions 选项函数类型
type GetOption func(*GetOptions)

// WithGetMessageLimit 设置加载的历史消息条数
// limit: 消息条数，0 表示加载全部历史消息
func WithGetMessageLimit(limit int) GetOption {
	return func(o *GetOptions) {
		o.messageLimit = limit
	}
}

// WithSkipWindowCheck 跳过读取时的窗口检查与紧急裁剪
//
// 用于历史贡献者通道：宿主在 preamble 就绪后会执行含 preamble 占用口径的
// mid-turn 压缩，此处再裁剪属于重复计算，且软阈值会重复调度后台摘要。
func WithSkipWindowCheck() GetOption {
	return func(o *GetOptions) {
		o.skipWindowCheck = true
	}
}

// defaultGetOptions 返回默认配置
func defaultGetOptions() *GetOptions {
	return &GetOptions{
		messageLimit: 100000, // 默认加载全部
	}
}
