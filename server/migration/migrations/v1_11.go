package migrations

import (
	milvusentity "mayfly-go/internal/milvus/domain/entity"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func V1_11() []*gormigrate.Migration {
	var migrations []*gormigrate.Migration
	migrations = append(migrations, V1_11_0()...)
	return migrations
}

func V1_11_0() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			ID: "v1.11.0",
			Migrate: func(tx *gorm.DB) error {
				tx.Exec(`INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1775967861, 0, 'lUgXhO96/', 1, 1, 'AI', '/ai', 20000000, '{"icon":"icon ai/ai","isKeepAlive":true,"routeName":"ai"}', 1, 'admin', 1, 'admin', '2026-04-12 12:24:22', '2026-04-12 17:08:47', 0, NULL)`)
				tx.Exec(`INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES (1775967903, 1775967861, 'lUgXhO96/VPfzI5pQ/', 1, 1, 'menu.aiAssistant', 'assistant', 1775967903, '{"icon":"icon ai/assistant","isKeepAlive":true,"routeName":"AiAssistant"}', 1, 'admin', 1, 'admin', '2026-04-12 12:25:03', '2026-04-12 17:05:49', 0, NULL)`)
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "20260420-v1.11.0_milvus",
			Migrate: func(tx *gorm.DB) error {
				tx.AutoMigrate(&milvusentity.Milvus{})

				tx.Exec("INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES (1775614966, 1775614920, 2, 1, 'milvus-保存实例', 'milvus:save', 1775614966, 'null', 12, 'admin', 12, 'admin', '2026-04-08 10:22:46', '2026-04-08 10:23:14', 'deKHISON/mqh1LvkC/', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES (1775614985, 1775614920, 2, 1, 'milvus-删除实例', 'milvus:del', 1775614985, 'null', 12, 'admin', 12, 'admin', '2026-04-08 10:23:06', '2026-04-08 10:23:06', 'deKHISON/Tx49ICTo/', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES (1775615036, 1756122788, 2, 1, 'Milvus', 'milvus:base', 1775615036, 'null', 12, 'admin', 12, 'admin', '2026-04-08 10:23:57', '2026-04-08 10:23:57', 'ocdrUNaa/n10hVYeh/', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES (1775615104, 1775615036, 2, 1, 'milvus-数据保存', 'milvus:data:save', 1775615104, 'null', 12, 'admin', 12, 'admin', '2026-04-08 10:25:05', '2026-04-08 10:25:05', 'ocdrUNaa/n10hVYeh/OXALVxRy/', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES (1775615134, 1775615036, 2, 1, 'milvus-数据删除', 'milvus:data:del', 1775615134, 'null', 12, 'admin', 12, 'admin', '2026-04-08 10:25:35', '2026-04-08 10:25:35', 'ocdrUNaa/n10hVYeh/y5uPJpkY/', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES (1775614920, 94, 2, 1, 'Milvus', 'milvus', 1775614920, 'null', 12, 'admin', 12, 'admin', '2026-04-08 10:22:01', '2026-04-08 10:23:42', 'deKHISON/', 0, NULL)")

				tx.Exec("INSERT INTO `t_sys_role_resource` ( `role_id`, `resource_id`, `creator_id`, `creator`, `create_time`, `is_deleted`, `delete_time`) VALUES (1, 1775614920, 12, 'admin', '2026-04-08 10:28:58', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_role_resource` ( `role_id`, `resource_id`, `creator_id`, `creator`, `create_time`, `is_deleted`, `delete_time`) VALUES ( 1, 1775614966, 12, 'admin', '2026-04-08 10:28:58', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_role_resource` ( `role_id`, `resource_id`, `creator_id`, `creator`, `create_time`, `is_deleted`, `delete_time`) VALUES ( 1, 1775614985, 12, 'admin', '2026-04-08 10:28:58', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_role_resource` ( `role_id`, `resource_id`, `creator_id`, `creator`, `create_time`, `is_deleted`, `delete_time`) VALUES ( 1, 1775615036, 12, 'admin', '2026-04-08 10:28:58', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_role_resource` ( `role_id`, `resource_id`, `creator_id`, `creator`, `create_time`, `is_deleted`, `delete_time`) VALUES ( 1, 1775615104, 12, 'admin', '2026-04-08 10:28:58', 0, NULL)")
				tx.Exec("INSERT INTO `t_sys_role_resource` ( `role_id`, `resource_id`, `creator_id`, `creator`, `create_time`, `is_deleted`, `delete_time`) VALUES ( 1, 1775615134, 12, 'admin', '2026-04-08 10:28:58', 0, NULL)")
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}
}
