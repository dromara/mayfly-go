package migrationx

import "gorm.io/gorm"

// NoopRollback 空回滚（迁移不支持回滚时统一使用）
func NoopRollback(tx *gorm.DB) error { return nil }
