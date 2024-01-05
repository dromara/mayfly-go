-- ----------------------------
-- Table structure for t_db_backup
-- ----------------------------
DROP TABLE IF EXISTS `t_db_backup`;
CREATE TABLE `t_db_backup` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(32) NOT NULL COMMENT '备份名称',
    `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
    `db_name` varchar(64) NOT NULL COMMENT '数据库名称',
    `repeated` tinyint(1) DEFAULT NULL COMMENT '是否重复执行',
    `interval` bigint(20) DEFAULT NULL COMMENT '备份周期',
    `start_time` datetime DEFAULT NULL COMMENT '首次备份时间',
    `enabled` tinyint(1) DEFAULT NULL COMMENT '是否启用',
    `last_status` tinyint(4) DEFAULT NULL COMMENT '上次备份状态',
    `last_result` varchar(256) DEFAULT NULL COMMENT '上次备份结果',
    `last_time` datetime DEFAULT NULL COMMENT '上次备份时间',
    `create_time` datetime DEFAULT NULL,
    `creator_id` bigint(20) unsigned DEFAULT NULL,
    `creator` varchar(32) DEFAULT NULL,
    `update_time` datetime DEFAULT NULL,
    `modifier_id` bigint(20) unsigned DEFAULT NULL,
    `modifier` varchar(32) DEFAULT NULL,
    `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_db_name` (`db_name`) USING BTREE,
    KEY `idx_db_instance_id` (`db_instance_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_backup_history
-- ----------------------------
DROP TABLE IF EXISTS `t_db_backup_history`;
CREATE TABLE `t_db_backup_history` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(64) NOT NULL COMMENT '历史备份名称',
    `db_backup_id` bigint(20) unsigned NOT NULL COMMENT '数据库备份ID',
    `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
    `db_name` varchar(64) NOT NULL COMMENT '数据库名称',
    `uuid` varchar(36) NOT NULL COMMENT '历史备份uuid',
    `binlog_file_name` varchar(32) DEFAULT NULL COMMENT 'BINLOG文件名',
    `binlog_sequence` bigint(20) DEFAULT NULL COMMENT 'BINLOG序列号',
    `binlog_position` bigint(20) DEFAULT NULL COMMENT 'BINLOG位置',
    `create_time` datetime DEFAULT NULL COMMENT '历史备份创建时间',
    `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_db_backup_id` (`db_backup_id`) USING BTREE,
    KEY `idx_db_instance_id` (`db_instance_id`) USING BTREE,
    KEY `idx_db_name` (`db_name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_restore
-- ----------------------------
DROP TABLE IF EXISTS `t_db_restore`;
CREATE TABLE `t_db_restore` (
     `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
     `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
     `db_name` varchar(64) NOT NULL COMMENT '数据库名称',
     `repeated` tinyint(1) DEFAULT NULL COMMENT '是否重复执行',
     `interval` bigint(20) DEFAULT NULL COMMENT '恢复周期',
     `start_time` datetime DEFAULT NULL COMMENT '首次恢复时间',
     `enabled` tinyint(1) DEFAULT NULL COMMENT '是否启用',
     `last_status` tinyint(4) DEFAULT NULL COMMENT '上次恢复状态',
     `last_result` varchar(256) DEFAULT NULL COMMENT '上次恢复结果',
     `last_time` datetime DEFAULT NULL COMMENT '上次恢复时间',
     `point_in_time` datetime DEFAULT NULL COMMENT '恢复时间点',
     `db_backup_id` bigint(20) unsigned DEFAULT NULL COMMENT '备份ID',
     `db_backup_history_id` bigint(20) unsigned DEFAULT NULL COMMENT '历史备份ID',
     `db_backup_history_name` varchar(64) DEFAULT NULL COMMENT '历史备份名称',
     `create_time` datetime DEFAULT NULL,
     `creator_id` bigint(20) unsigned DEFAULT NULL,
     `creator` varchar(32) DEFAULT NULL,
     `update_time` datetime DEFAULT NULL,
     `modifier_id` bigint(20) unsigned DEFAULT NULL,
     `modifier` varchar(32) DEFAULT NULL,
     `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
     `delete_time` datetime DEFAULT NULL,
     PRIMARY KEY (`id`),
     KEY `idx_db_instane_id` (`db_instance_id`) USING BTREE,
     KEY `idx_db_name` (`db_name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=22 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_restore_history
-- ----------------------------
DROP TABLE IF EXISTS `t_db_restore_history`;
CREATE TABLE `t_db_restore_history` (
     `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
     `db_restore_id` bigint(20) unsigned NOT NULL COMMENT '恢复ID',
     `create_time` datetime DEFAULT NULL COMMENT '历史恢复创建时间',
     `is_deleted` tinyint(4) NOT NULL DEFAULT 0,
     `delete_time` datetime DEFAULT NULL,
     PRIMARY KEY (`id`),
     KEY `idx_db_restore_id` (`db_restore_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_binlog
-- ----------------------------
DROP TABLE IF EXISTS `t_db_binlog`;
CREATE TABLE `t_db_binlog` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
    `last_status` bigint(20) DEFAULT NULL COMMENT '上次下载状态',
    `last_result` varchar(256) DEFAULT NULL COMMENT '上次下载结果',
    `last_time` datetime DEFAULT NULL COMMENT '上次下载时间',
    `create_time` datetime DEFAULT NULL,
    `creator_id` bigint(20) unsigned DEFAULT NULL,
    `creator` varchar(32) DEFAULT NULL,
    `update_time` datetime DEFAULT NULL,
    `modifier_id` bigint(20) unsigned DEFAULT NULL,
    `modifier` varchar(32) DEFAULT NULL,
    `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_db_instance_id` (`db_instance_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for t_db_binlog_history
-- ----------------------------
DROP TABLE IF EXISTS `t_db_binlog_history`;
CREATE TABLE `t_db_binlog_history` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `db_instance_id` bigint(20) unsigned NOT NULL COMMENT '数据库实例ID',
    `file_name` varchar(32) DEFAULT NULL COMMENT 'BINLOG文件名称',
    `file_size` bigint(20) DEFAULT NULL COMMENT 'BINLOG文件大小',
    `sequence` bigint(20) DEFAULT NULL COMMENT 'BINLOG序列号',
    `first_event_time` datetime DEFAULT NULL COMMENT '首次事件时间',
    `create_time` datetime DEFAULT NULL,
    `is_deleted` tinyint(4) NOT NULL DEFAULT 0,
    `delete_time` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_db_instance_id` (`db_instance_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('Mysql可执行文件', 'MysqlBin', '[{"model":"path","name":"路径","placeholder":"可执行文件路径","required":true},{"model":"mysql","name":"mysql","placeholder":"mysql命令路径(空则为 路径/mysql)","required":false},{"model":"mysqldump","name":"mysqldump","placeholder":"mysqldump命令路径(空则为 路径/mysqldump)","required":false},{"model":"mysqlbinlog","name":"mysqlbinlog","placeholder":"mysqlbinlog命令路径(空则为 路径/mysqlbinlog)","required":false}]', '{"mysql":"","mysqldump":"","mysqlbinlog":"","path":""}', '', 'admin,', '2023-12-29 10:01:33', 1, 'admin', '2023-12-29 13:34:40', 1, 'admin', 0, NULL);
INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('数据库备份恢复', 'DbBackupRestore', '[{"model":"backupPath","name":"备份路径","placeholder":"备份文件存储路径"}]', '{"backupPath":"./db/backup"}', '', 'admin,', '2023-12-29 09:55:26', 1, 'admin', '2023-12-29 15:45:24', 1, 'admin', 0, NULL);
DELETE FROM `t_sys_config` WHERE `key` = 'UseWatermark'
INSERT INTO `t_sys_config` (`name`, `key`, `params`, `value`, `remark`, `permission`, `create_time`, `creator_id`, `creator`, `update_time`, `modifier_id`, `modifier`, `is_deleted`, `delete_time`) VALUES('系统全局样式设置', 'SysStyleConfig', '[{"model":"logoIcon","name":"logo图标","placeholder":"系统logo图标（base64编码, 建议svg格式，不超过10k）","required":false},{"model":"title","name":"菜单栏标题","placeholder":"系统菜单栏标题展示","required":false},{"model":"viceTitle","name":"登录页标题","placeholder":"登录页标题展示","required":false},{"model":"useWatermark","name":"是否启用水印","placeholder":"是否启用系统水印","options":"true,false","required":false},{"model":"watermarkContent","name":"水印补充信息","placeholder":"额外水印信息","required":false}]', '{"title":"mayfly-go","viceTitle":"mayfly-go","logoIcon":"","useWatermark":"true","watermarkContent":""}', '系统icon、标题、水印信息等配置', 'all', '2024-01-04 15:17:18', 1, 'admin', '2024-01-05 09:40:44', 1, 'admin', 0, NULL);