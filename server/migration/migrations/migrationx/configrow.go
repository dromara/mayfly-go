package migrationx

import (
	sysentity "mayfly-go/internal/sys/domain/entity"

	"gorm.io/gorm"
)

// ConfigRowSpec 单例配置行的归一口径
type ConfigRowSpec struct {
	Key        string // 配置键（t_sys_config.key）
	Name       string // 配置显示名（i18n key），仅在该行 name 为空时补齐
	Permission string // 归一到「可操作该配置」的权限口径（如 admin,）
	// DefaultValue 行不存在时的建行初始值
	DefaultValue string
}

// NormalizeSingleConfigRow 把某个配置键收敛为「恰好一条存活行」，并归一权限口径。
//
// 用于修复「配置行权限为空 → 管理页按权限过滤看不到 → 每次保存都新增一行」这类
// 数据缺陷：重复行会让后端按 key 取首行读到旧值（改动静默不生效），且随时间无限增长。
//
// 收敛规则（幂等，重复执行安全）：
//   - 无存活行：按 spec 建一行，权限与显示名直接落地
//   - 有存活行：保留 id 最小的一行（后端按 key 取首行的稳定目标），取值继承 id 最大
//     的一行（最新一次保存写的是全量值，继承它才不会丢掉用户在故障期间改过的内容），
//     权限与空显示名一并归一；其余行软删
func NormalizeSingleConfigRow(tx *gorm.DB, spec ConfigRowSpec) error {
	// 迁移层直接用 gorm，没有基座的 is_deleted = 0 自动条件：必须显式过滤，
	// 否则历史软删行会被当成存活行参与「保留/继承最新值」的判定
	var rows []sysentity.Config
	if err := tx.Where("`key` = ? AND is_deleted = 0", spec.Key).Order("id ASC").Find(&rows).Error; err != nil {
		return err
	}

	if len(rows) == 0 {
		row := &sysentity.Config{
			Name:       spec.Name,
			Key:        spec.Key,
			Value:      spec.DefaultValue,
			Permission: spec.Permission,
		}
		// 迁移无操作者上下文：走基座的系统账号兜底（create_time / creator 等 NOT NULL 列
		// 必须被填上，否则严格模式下插入直接报错）
		row.FillBaseInfo(nil)
		return tx.Create(row).Error
	}

	keep := rows[0]
	updates := map[string]any{"permission": spec.Permission}
	// 最新一行可能正是 keep 本身；只在确有更新值时覆盖，避免把已修好的值清空
	if latest := rows[len(rows)-1].Value; latest != "" && latest != keep.Value {
		updates["value"] = latest
	}
	if keep.Name == "" && spec.Name != "" {
		updates["name"] = spec.Name
	}
	if err := tx.Model(&sysentity.Config{}).Where("id = ?", keep.Id).Updates(updates).Error; err != nil {
		return err
	}

	duplicatedIds := make([]uint64, 0, len(rows)-1)
	for _, row := range rows[1:] {
		duplicatedIds = append(duplicatedIds, row.Id)
	}
	if len(duplicatedIds) == 0 {
		return nil
	}
	// 软删而非物理删：配置行属可审计数据，软删后仍可回溯，且不影响按 key 的读取
	return tx.Model(&sysentity.Config{}).Where("id IN ?", duplicatedIds).Update("is_deleted", 1).Error
}
