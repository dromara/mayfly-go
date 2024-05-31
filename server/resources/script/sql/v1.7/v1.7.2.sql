INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(160, 49, 'dbms23ax/xleaiec2/3NUXQFIO/', 2, 1, '数据库备份', 'db:backup', 1705973876, 'null', 1, 'admin', 1, 'admin', '2024-01-23 09:37:56', '2024-01-23 09:37:56', 0, NULL);
INSERT INTO t_sys_resource (id, pid, ui_path, `type`, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) VALUES(161, 49, 'dbms23ax/xleaiec2/ghErkTdb/', 2, 1, '数据库恢复', 'db:restore', 1705973909, 'null', 1, 'admin', 1, 'admin', '2024-01-23 09:38:29', '2024-01-23 09:38:29', 0, NULL);

ALTER TABLE `t_db_backup`
    ADD COLUMN `enabled_desc` varchar(64) NULL COMMENT '任务启用描述' AFTER `enabled`;

ALTER TABLE `t_db_restore`
    ADD COLUMN `enabled_desc` varchar(64) NULL COMMENT '任务启用描述' AFTER `enabled`;

ALTER TABLE `t_db_backup_history`
    ADD COLUMN `restoring` tinyint(1) NOT NULL DEFAULT '0' COMMENT '备份历史恢复标识',
    ADD COLUMN `deleting` tinyint(1) NOT NULL DEFAULT '0' COMMENT '备份历史删除标识';
