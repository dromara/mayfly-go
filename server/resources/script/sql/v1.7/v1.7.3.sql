ALTER TABLE `t_db_backup`
    ADD COLUMN `max_save_days` int(8) NOT NULL DEFAULT '0' COMMENT '最大保存天数' AFTER `interval`;

ALTER TABLE `t_db_binlog_history`
    ADD COLUMN `last_event_time` datetime NULL DEFAULT NULL COMMENT '最新事件时间' AFTER `first_event_time`;

INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1707206386, 2, 1, 1, '机器操作', 'machines-op', 1, '{"component":"ops/machine/MachineOp","icon":"Monitor","isKeepAlive":true,"routeName":"MachineOp"}', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-06 15:59:46', '2024-02-06 16:24:21', 'PDPt6217/', 0, NULL);
INSERT INTO `t_sys_resource` (`id`, `pid`, `type`, `status`, `name`, `code`, `weight`, `meta`, `creator_id`, `creator`, `modifier_id`, `modifier`, `create_time`, `update_time`, `ui_path`, `is_deleted`, `delete_time`) VALUES(1707206421, 1707206386, 2, 1, '基本权限', 'machine-op', 1707206421, 'null', 12, 'liuzongyang', 12, 'liuzongyang', '2024-02-06 16:00:22', '2024-02-06 16:00:22', 'PDPt6217/kQXTYvuM/', 0, NULL);