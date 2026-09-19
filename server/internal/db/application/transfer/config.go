package transfer

// 迁移模块集中配置常量。
// 避免魔法数字散落在各文件中，便于统一调优和维护。

const (
	// DefaultTransferConcurrency 默认迁移并行度
	DefaultTransferConcurrency = 4
	// MinTransferConcurrency 最小并行度
	MinTransferConcurrency = 1
	// MaxTransferConcurrency 最大并行度（防止过大并行拖垮源/目标库）
	MaxTransferConcurrency = 16

	// TransferLogCacheTTL 迁移日志缓存过期时间（秒）
	TransferLogCacheTTLSeconds = 3600 // 1 hour
)
